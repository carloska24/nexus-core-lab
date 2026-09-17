package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/carloska24/nexus-core-lab/internal/device"
	"github.com/carloska24/nexus-core-lab/internal/network"
	"github.com/carloska24/nexus-core-lab/internal/platform/httpserver"
	"github.com/carloska24/nexus-core-lab/internal/platform/storage"
	"github.com/carloska24/nexus-core-lab/internal/session"
	"github.com/carloska24/nexus-core-lab/internal/subscriber"
	"github.com/carloska24/nexus-core-lab/internal/telemetry"
)

const (
	publicDemoCookieName = "nexus_demo_session"
	demoRateWindow       = time.Minute
	demoRequestBodyLimit = int64(64 << 10)
)

var errPublicDemoCapacity = errors.New("public demo visitor capacity reached")

type publicDemoConfig struct {
	Enabled              bool
	IdleTTL              time.Duration
	AbsoluteTTL          time.Duration
	MaxContexts          int
	MaxSubscribers       int
	MaxDevices           int
	MaxConnectedSessions int
	MutationsPerWindow   int
	ResetsPerWindow      int
	RateWindow           time.Duration
	RequestBodyLimit     int64
}

func defaultPublicDemoConfig() publicDemoConfig {
	return publicDemoConfig{
		IdleTTL:              30 * time.Minute,
		AbsoluteTTL:          2 * time.Hour,
		MaxContexts:          100,
		MaxSubscribers:       20,
		MaxDevices:           20,
		MaxConnectedSessions: 10,
		MutationsPerWindow:   30,
		ResetsPerWindow:      3,
		RateWindow:           demoRateWindow,
		RequestBodyLimit:     demoRequestBodyLimit,
	}
}

func loadPublicDemoConfig() (publicDemoConfig, error) {
	cfg := defaultPublicDemoConfig()
	var err error
	if raw := strings.TrimSpace(os.Getenv("PUBLIC_DEMO_MODE")); raw != "" {
		cfg.Enabled, err = strconv.ParseBool(raw)
		if err != nil {
			return cfg, fmt.Errorf("PUBLIC_DEMO_MODE must be a boolean: %w", err)
		}
	}
	if !cfg.Enabled {
		return cfg, nil
	}

	if cfg.IdleTTL, err = envDuration("PUBLIC_DEMO_IDLE_TTL", cfg.IdleTTL); err != nil {
		return cfg, err
	}
	if cfg.AbsoluteTTL, err = envDuration("PUBLIC_DEMO_ABSOLUTE_TTL", cfg.AbsoluteTTL); err != nil {
		return cfg, err
	}
	if cfg.MaxContexts, err = envPositiveInt("PUBLIC_DEMO_MAX_CONTEXTS", cfg.MaxContexts); err != nil {
		return cfg, err
	}
	if cfg.MaxSubscribers, err = envPositiveInt("PUBLIC_DEMO_MAX_SUBSCRIBERS", cfg.MaxSubscribers); err != nil {
		return cfg, err
	}
	if cfg.MaxDevices, err = envPositiveInt("PUBLIC_DEMO_MAX_DEVICES", cfg.MaxDevices); err != nil {
		return cfg, err
	}
	if cfg.MaxConnectedSessions, err = envPositiveInt("PUBLIC_DEMO_MAX_CONNECTED_SESSIONS", cfg.MaxConnectedSessions); err != nil {
		return cfg, err
	}
	if cfg.MutationsPerWindow, err = envPositiveInt("PUBLIC_DEMO_MUTATIONS_PER_MINUTE", cfg.MutationsPerWindow); err != nil {
		return cfg, err
	}
	if cfg.ResetsPerWindow, err = envPositiveInt("PUBLIC_DEMO_RESETS_PER_MINUTE", cfg.ResetsPerWindow); err != nil {
		return cfg, err
	}
	if cfg.AbsoluteTTL < cfg.IdleTTL {
		return cfg, errors.New("PUBLIC_DEMO_ABSOLUTE_TTL must be greater than or equal to PUBLIC_DEMO_IDLE_TTL")
	}
	return cfg, nil
}

func envDuration(name string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", name)
	}
	return value, nil
}

func envPositiveInt(name string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return value, nil
}

type fixedWindowLimiter struct {
	mu          sync.Mutex
	windowStart time.Time
	count       int
}

func (l *fixedWindowLimiter) allow(now time.Time, limit int, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.windowStart.IsZero() || now.Sub(l.windowStart) >= window {
		l.windowStart = now
		l.count = 0
	}
	if l.count >= limit {
		return false
	}
	l.count++
	return true
}

type publicDemoContext struct {
	createdAt time.Time
	lastSeen  time.Time
	handler   http.Handler
	worker    *telemetry.Worker

	subscriberRepo subscriber.Repository
	deviceRepo     device.Repository
	sessionRepo    session.Repository

	mutationMu      sync.Mutex
	mutationLimiter fixedWindowLimiter
	resetLimiter    *fixedWindowLimiter
}

func newPublicDemoContext(now time.Time) *publicDemoContext {
	subscriberRepo := subscriber.NewMemoryRepository()
	deviceRepo := device.NewMemoryRepository()
	sessionRepo := session.NewMemoryRepository()
	ipPool := network.NewIPPool()
	worker := telemetry.NewWorker()
	worker.Start()

	subscriberService := subscriber.NewService(subscriberRepo)
	deviceService := device.NewService(deviceRepo, &subscriberCheckerAdapter{subService: subscriberService})
	sessionService := session.NewService(sessionRepo, &deviceCheckerAdapter{
		deviceService: deviceService,
		subService:    subscriberService,
	}, ipPool, &telemetrySessionAdapter{worker: worker})

	var requestsCounter atomic.Uint64
	router := httpserver.New(
		subscriber.NewHandler(subscriberService).RegisterRoutes,
		device.NewHandler(deviceService).RegisterRoutes,
		session.NewHandler(sessionService).RegisterRoutes,
		telemetry.NewHandler(worker.Metrics(), sessionRepo, &requestsCounter).RegisterRoutes,
		worker.RegisterRecentRoutes,
		network.NewIPPoolHandler(ipPool).RegisterRoutes,
		storage.NewHandler(nil).RegisterRoutes,
	)

	return &publicDemoContext{
		createdAt:      now,
		lastSeen:       now,
		handler:        httpserver.TelemetryMiddleware(&requestsCounter)(router),
		worker:         worker,
		subscriberRepo: subscriberRepo,
		deviceRepo:     deviceRepo,
		sessionRepo:    sessionRepo,
		resetLimiter:   &fixedWindowLimiter{},
	}
}

func (c *publicDemoContext) close(ctx context.Context) error {
	return c.worker.Shutdown(ctx)
}

func (c *publicDemoContext) expired(now time.Time, cfg publicDemoConfig) bool {
	return now.Sub(c.lastSeen) >= cfg.IdleTTL || now.Sub(c.createdAt) >= cfg.AbsoluteTTL
}

func (c *publicDemoContext) serveHTTP(w http.ResponseWriter, r *http.Request, cfg publicDemoConfig, now time.Time) {
	if !isMutation(r.Method) {
		c.handler.ServeHTTP(w, r)
		return
	}

	c.mutationMu.Lock()
	defer c.mutationMu.Unlock()
	if !c.mutationLimiter.allow(now, cfg.MutationsPerWindow, cfg.RateWindow) {
		writePublicDemoError(w, http.StatusTooManyRequests, "DEMO_RATE_LIMIT", "public demo mutation rate limit reached")
		return
	}
	if message, limited := c.resourceLimit(r, cfg); limited {
		writePublicDemoError(w, http.StatusTooManyRequests, "DEMO_RESOURCE_LIMIT", message)
		return
	}
	c.handler.ServeHTTP(w, r)
}

func (c *publicDemoContext) resourceLimit(r *http.Request, cfg publicDemoConfig) (string, bool) {
	ctx := r.Context()
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/subscribers":
		items, err := c.subscriberRepo.List(ctx, subscriber.ListFilter{})
		return fmt.Sprintf("public demo subscriber limit reached (%d)", cfg.MaxSubscribers), err == nil && len(items) >= cfg.MaxSubscribers
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/devices":
		items, err := c.deviceRepo.List(ctx)
		return fmt.Sprintf("public demo device limit reached (%d)", cfg.MaxDevices), err == nil && len(items) >= cfg.MaxDevices
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sessions/attach":
		count, err := c.sessionRepo.ActiveCount(ctx)
		return fmt.Sprintf("public demo connected session limit reached (%d)", cfg.MaxConnectedSessions), err == nil && count >= cfg.MaxConnectedSessions
	default:
		return "", false
	}
}

type publicDemoRegistry struct {
	mu       sync.Mutex
	contexts map[string]*publicDemoContext
	config   publicDemoConfig
	now      func() time.Time
	health   http.Handler
}

func newPublicDemoRegistry(cfg publicDemoConfig) *publicDemoRegistry {
	return &publicDemoRegistry{
		contexts: make(map[string]*publicDemoContext),
		config:   cfg,
		now:      time.Now,
		health:   httpserver.New(),
	}
}

func (r *publicDemoRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodGet && req.URL.Path == "/health" {
		r.health.ServeHTTP(w, req)
		return
	}

	ctx, token, setCookie, expired, err := r.resolve(req)
	for _, old := range expired {
		shutdownDemoContext(old)
	}
	if err != nil {
		status := http.StatusInternalServerError
		code := "DEMO_CONTEXT_ERROR"
		message := "failed to initialize public demo context"
		if errors.Is(err, errPublicDemoCapacity) {
			status, code, message = http.StatusServiceUnavailable, "DEMO_CAPACITY_REACHED", err.Error()
		}
		writePublicDemoError(w, status, code, message)
		return
	}
	if setCookie {
		writePublicDemoCookie(w, req, token, r.config.AbsoluteTTL)
	}

	now := r.now()
	if req.Method == http.MethodPost && req.URL.Path == "/api/v1/demo/reset" {
		if !ctx.resetLimiter.allow(now, r.config.ResetsPerWindow, r.config.RateWindow) {
			writePublicDemoError(w, http.StatusTooManyRequests, "DEMO_RESET_RATE_LIMIT", "public demo reset rate limit reached")
			return
		}
		r.reset(token, ctx, now)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "reset"})
		return
	}

	ctx.serveHTTP(w, req, r.config, now)
}

func (r *publicDemoRegistry) resolve(req *http.Request) (*publicDemoContext, string, bool, []*publicDemoContext, error) {
	now := r.now()
	token := requestDemoToken(req)

	r.mu.Lock()
	expired := r.removeExpiredLocked(now)
	if token != "" {
		if existing, ok := r.contexts[token]; ok {
			existing.lastSeen = now
			r.mu.Unlock()
			return existing, token, false, expired, nil
		}
	}
	if len(r.contexts) >= r.config.MaxContexts {
		r.mu.Unlock()
		return nil, "", false, expired, errPublicDemoCapacity
	}
	var err error
	for {
		token, err = newDemoToken()
		if err != nil {
			r.mu.Unlock()
			return nil, "", false, expired, err
		}
		if _, collision := r.contexts[token]; !collision {
			break
		}
	}
	created := newPublicDemoContext(now)
	r.contexts[token] = created
	r.mu.Unlock()
	return created, token, true, expired, nil
}

func (r *publicDemoRegistry) removeExpiredLocked(now time.Time) []*publicDemoContext {
	var expired []*publicDemoContext
	for token, ctx := range r.contexts {
		if ctx.expired(now, r.config) {
			delete(r.contexts, token)
			expired = append(expired, ctx)
		}
	}
	return expired
}

func (r *publicDemoRegistry) reset(token string, current *publicDemoContext, now time.Time) {
	replacement := newPublicDemoContext(now)
	replacement.resetLimiter = current.resetLimiter
	r.mu.Lock()
	if r.contexts[token] != current {
		r.mu.Unlock()
		shutdownDemoContext(replacement)
		return
	}
	r.contexts[token] = replacement
	r.mu.Unlock()
	shutdownDemoContext(current)
}

func (r *publicDemoRegistry) Shutdown(ctx context.Context) error {
	r.mu.Lock()
	contexts := make([]*publicDemoContext, 0, len(r.contexts))
	for token, item := range r.contexts {
		delete(r.contexts, token)
		contexts = append(contexts, item)
	}
	r.mu.Unlock()

	var firstErr error
	for _, item := range contexts {
		if err := item.close(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func shutdownDemoContext(item *publicDemoContext) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = item.close(ctx)
}

func requestDemoToken(req *http.Request) string {
	cookie, err := req.Cookie(publicDemoCookieName)
	if err != nil {
		return ""
	}
	decoded, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil || len(decoded) != 16 {
		return ""
	}
	return cookie.Value
}

func newDemoToken() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate public demo token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func writePublicDemoCookie(w http.ResponseWriter, req *http.Request, token string, lifetime time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     publicDemoCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(lifetime.Seconds()),
		HttpOnly: true,
		Secure:   requestUsesHTTPS(req),
		SameSite: http.SameSiteStrictMode,
	})
}

func requestUsesHTTPS(req *http.Request) bool {
	if req.TLS != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(strings.Split(req.Header.Get("X-Forwarded-Proto"), ",")[0]), "https")
}

func publicDemoSafety(next http.Handler, cfg publicDemoConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setPublicDemoSecurityHeaders(w)
		if isMutation(r.Method) {
			if !validMutationOrigin(r) {
				writePublicDemoError(w, http.StatusForbidden, "INVALID_ORIGIN", "cross-origin mutations are not allowed")
				return
			}
			if r.ContentLength > cfg.RequestBodyLimit {
				writePublicDemoError(w, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "request body exceeds public demo limit")
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, cfg.RequestBodyLimit)
			if requiresJSONBody(r) && !hasJSONContentType(r.Header.Get("Content-Type")) {
				writePublicDemoError(w, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "Content-Type application/json is required")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func setPublicDemoSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self' https://*.openfreemap.org; img-src 'self' data: blob: https://*.openfreemap.org; style-src 'self' 'unsafe-inline'; script-src 'self'; font-src 'self' data:; worker-src 'self' blob:; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Permissions-Policy", "geolocation=(), camera=(), microphone=()")
}

func isMutation(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
}

func requiresJSONBody(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	path := r.URL.Path
	return path == "/api/v1/subscribers" || path == "/api/v1/devices" || path == "/api/v1/sessions/attach" || strings.HasSuffix(path, "/handover")
}

func hasJSONContentType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	return err == nil && mediaType == "application/json"
}

func validMutationOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		return false
	}
	requestHost := r.Host
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Host"), ",")[0]); forwarded != "" {
		requestHost = forwarded
	}
	if strings.EqualFold(parsed.Host, requestHost) {
		return true
	}
	return isLoopbackHostname(parsed.Hostname()) && isLoopbackHostname(hostnameOnly(requestHost))
}

func hostnameOnly(hostport string) string {
	if parsed, err := url.Parse("http://" + hostport); err == nil {
		return parsed.Hostname()
	}
	return hostport
}

func isLoopbackHostname(host string) bool {
	return strings.EqualFold(host, "localhost") || host == "127.0.0.1" || host == "::1"
}

func writePublicDemoError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   strings.ToLower(code),
		"message": message,
		"code":    code,
	})
}
