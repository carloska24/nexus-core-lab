package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/carloska24/nexus-core-lab/internal/device"
	"github.com/carloska24/nexus-core-lab/internal/network"
	"github.com/carloska24/nexus-core-lab/internal/session"
	"github.com/carloska24/nexus-core-lab/internal/subscriber"
	"github.com/carloska24/nexus-core-lab/internal/telemetry"
)

type demoHTTPClient struct {
	client *http.Client
	base   string
}

func newDemoTestServer(t *testing.T, cfg publicDemoConfig) (*publicDemoRegistry, string) {
	t.Helper()
	registry := newPublicDemoRegistry(cfg)
	server := httptest.NewServer(publicDemoSafety(registry, cfg))
	t.Cleanup(func() {
		server.Close()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := registry.Shutdown(ctx); err != nil {
			t.Errorf("registry shutdown: %v", err)
		}
	})
	return registry, server.URL
}

func newDemoHTTPClient(t *testing.T, base string) *demoHTTPClient {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &demoHTTPClient{client: &http.Client{Jar: jar}, base: base}
}

func (c *demoHTTPClient) request(t *testing.T, method, path string, body any) (*http.Response, []byte) {
	t.Helper()
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		payload = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, c.base+path, payload)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	return resp, data
}

func decodeDemoJSON[T any](t *testing.T, data []byte) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("decode response %q: %v", data, err)
	}
	return value
}

func requireDemoStatus(t *testing.T, resp *http.Response, data []byte, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Fatalf("status=%d want=%d body=%s", resp.StatusCode, want, data)
	}
}

func provisionDemoSubscriber(t *testing.T, client *demoHTTPClient, suffix string) *subscriber.Subscriber {
	t.Helper()
	resp, data := client.request(t, http.MethodPost, "/api/v1/subscribers", map[string]string{
		"imsi":   "72400" + suffix,
		"msisdn": "+55" + suffix,
	})
	requireDemoStatus(t, resp, data, http.StatusCreated)
	return decodeDemoJSON[*subscriber.Subscriber](t, data)
}

func activateDemoSubscriber(t *testing.T, client *demoHTTPClient, id string) {
	t.Helper()
	resp, data := client.request(t, http.MethodPost, "/api/v1/subscribers/"+id+"/activate", nil)
	requireDemoStatus(t, resp, data, http.StatusOK)
}

func registerDemoDevice(t *testing.T, client *demoHTTPClient, subscriberID, suffix string) *device.Device {
	t.Helper()
	resp, data := client.request(t, http.MethodPost, "/api/v1/devices", map[string]string{
		"subscriber_id": subscriberID,
		"imei":          "86000" + suffix,
		"technology":    "5G",
	})
	requireDemoStatus(t, resp, data, http.StatusCreated)
	return decodeDemoJSON[*device.Device](t, data)
}

func attachDemoSession(t *testing.T, client *demoHTTPClient, deviceID, cellID string) *session.Session {
	t.Helper()
	resp, data := client.request(t, http.MethodPost, "/api/v1/sessions/attach", map[string]string{
		"device_id": deviceID,
		"cell_id":   cellID,
	})
	requireDemoStatus(t, resp, data, http.StatusCreated)
	return decodeDemoJSON[*session.Session](t, data)
}

func eventuallyDemo(t *testing.T, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition was not satisfied")
}

func TestPublicDemoHeroFlowAndVisitorIsolation(t *testing.T) {
	cfg := defaultPublicDemoConfig()
	_, base := newDemoTestServer(t, cfg)
	visitorA := newDemoHTTPClient(t, base)
	visitorB := newDemoHTTPClient(t, base)

	subA := provisionDemoSubscriber(t, visitorA, "0000000001")
	resp, data := visitorB.request(t, http.MethodGet, "/api/v1/subscribers", nil)
	requireDemoStatus(t, resp, data, http.StatusOK)
	if got := decodeDemoJSON[[]*subscriber.Subscriber](t, data); len(got) != 0 {
		t.Fatalf("visitor B saw visitor A subscribers: %d", len(got))
	}

	activateDemoSubscriber(t, visitorA, subA.ID)
	devA := registerDemoDevice(t, visitorA, subA.ID, "0000000001")
	resp, data = visitorB.request(t, http.MethodGet, "/api/v1/devices", nil)
	requireDemoStatus(t, resp, data, http.StatusOK)
	if got := decodeDemoJSON[[]*device.Device](t, data); len(got) != 0 {
		t.Fatalf("visitor B saw visitor A devices: %d", len(got))
	}

	sessA := attachDemoSession(t, visitorA, devA.ID, "CELL-SP-001")
	resp, data = visitorB.request(t, http.MethodGet, "/api/v1/sessions?device_id="+devA.ID, nil)
	requireDemoStatus(t, resp, data, http.StatusNotFound)

	eventuallyDemo(t, func() bool {
		resp, data := visitorA.request(t, http.MethodGet, "/api/v1/events/recent", nil)
		return resp.StatusCode == http.StatusOK && len(decodeDemoJSON[[]telemetry.Event](t, data)) == 1
	})
	resp, data = visitorB.request(t, http.MethodGet, "/api/v1/events/recent", nil)
	requireDemoStatus(t, resp, data, http.StatusOK)
	if got := decodeDemoJSON[[]telemetry.Event](t, data); len(got) != 0 {
		t.Fatalf("visitor B saw visitor A events: %d", len(got))
	}

	resp, data = visitorA.request(t, http.MethodGet, "/telemetry", nil)
	requireDemoStatus(t, resp, data, http.StatusOK)
	teleA := decodeDemoJSON[telemetry.TelemetryResponse](t, data)
	if teleA.Metrics.ActiveSessions != 1 || teleA.EventsTotal.Attach != 1 {
		t.Fatalf("visitor A telemetry mismatch: %+v", teleA)
	}
	resp, data = visitorB.request(t, http.MethodGet, "/telemetry", nil)
	requireDemoStatus(t, resp, data, http.StatusOK)
	teleB := decodeDemoJSON[telemetry.TelemetryResponse](t, data)
	if teleB.Metrics.ActiveSessions != 0 || teleB.EventsTotal.Attach != 0 {
		t.Fatalf("visitor B telemetry leaked: %+v", teleB)
	}

	resp, data = visitorA.request(t, http.MethodGet, "/api/v1/network/ip-pool", nil)
	requireDemoStatus(t, resp, data, http.StatusOK)
	if pool := decodeDemoJSON[network.IPPoolSnapshot](t, data); pool.Allocated != 1 {
		t.Fatalf("visitor A pool allocated=%d", pool.Allocated)
	}
	resp, data = visitorB.request(t, http.MethodGet, "/api/v1/network/ip-pool", nil)
	requireDemoStatus(t, resp, data, http.StatusOK)
	if pool := decodeDemoJSON[network.IPPoolSnapshot](t, data); pool.Allocated != 0 {
		t.Fatalf("visitor B pool leaked allocation=%d", pool.Allocated)
	}

	resp, data = visitorA.request(t, http.MethodPost, "/api/v1/sessions/"+sessA.ID+"/handover", map[string]string{"target_cell_id": "CELL-SP-002"})
	requireDemoStatus(t, resp, data, http.StatusOK)
	resp, data = visitorA.request(t, http.MethodPost, "/api/v1/sessions/"+sessA.ID+"/detach", nil)
	requireDemoStatus(t, resp, data, http.StatusOK)
	eventuallyDemo(t, func() bool {
		_, data := visitorA.request(t, http.MethodGet, "/api/v1/events/recent", nil)
		return len(decodeDemoJSON[[]telemetry.Event](t, data)) == 3
	})
}

func TestPublicDemoResetIsIsolatedAndIdempotent(t *testing.T) {
	cfg := defaultPublicDemoConfig()
	_, base := newDemoTestServer(t, cfg)
	visitorA, visitorB := newDemoHTTPClient(t, base), newDemoHTTPClient(t, base)
	provisionDemoSubscriber(t, visitorA, "0000000011")
	provisionDemoSubscriber(t, visitorB, "0000000012")

	for range 2 {
		resp, data := visitorA.request(t, http.MethodPost, "/api/v1/demo/reset", nil)
		requireDemoStatus(t, resp, data, http.StatusOK)
	}
	_, data := visitorA.request(t, http.MethodGet, "/api/v1/subscribers", nil)
	if got := decodeDemoJSON[[]*subscriber.Subscriber](t, data); len(got) != 0 {
		t.Fatalf("visitor A reset retained %d subscribers", len(got))
	}
	_, data = visitorB.request(t, http.MethodGet, "/api/v1/subscribers", nil)
	if got := decodeDemoJSON[[]*subscriber.Subscriber](t, data); len(got) != 1 {
		t.Fatalf("visitor B changed by visitor A reset: %d", len(got))
	}
}

type controlledDemoClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *controlledDemoClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *controlledDemoClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
}

func TestPublicDemoIdleAndAbsoluteExpiration(t *testing.T) {
	cases := []struct {
		name       string
		idle       time.Duration
		absolute   time.Duration
		advances   []time.Duration
		touchAfter bool
	}{
		{"idle", 10 * time.Minute, time.Hour, []time.Duration{11 * time.Minute}, false},
		{"absolute", 10 * time.Minute, 25 * time.Minute, []time.Duration{8 * time.Minute, 8 * time.Minute, 10 * time.Minute}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := defaultPublicDemoConfig()
			cfg.IdleTTL, cfg.AbsoluteTTL = tc.idle, tc.absolute
			registry, base := newDemoTestServer(t, cfg)
			clock := &controlledDemoClock{now: time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)}
			registry.now = clock.Now
			client := newDemoHTTPClient(t, base)
			provisionDemoSubscriber(t, client, "0000000021")
			for i, advance := range tc.advances {
				clock.Advance(advance)
				if tc.touchAfter && i < len(tc.advances)-1 {
					resp, data := client.request(t, http.MethodGet, "/api/v1/subscribers", nil)
					requireDemoStatus(t, resp, data, http.StatusOK)
				}
			}
			_, data := client.request(t, http.MethodGet, "/api/v1/subscribers", nil)
			if got := decodeDemoJSON[[]*subscriber.Subscriber](t, data); len(got) != 0 {
				t.Fatalf("expired context retained %d subscribers", len(got))
			}
		})
	}
}

func TestPublicDemoContextCap(t *testing.T) {
	cfg := defaultPublicDemoConfig()
	cfg.MaxContexts = 1
	_, base := newDemoTestServer(t, cfg)
	first, second := newDemoHTTPClient(t, base), newDemoHTTPClient(t, base)
	resp, data := first.request(t, http.MethodGet, "/api/v1/subscribers", nil)
	requireDemoStatus(t, resp, data, http.StatusOK)
	resp, data = second.request(t, http.MethodGet, "/api/v1/subscribers", nil)
	requireDemoStatus(t, resp, data, http.StatusServiceUnavailable)
}

func TestPublicDemoResourceLimits(t *testing.T) {
	t.Run("subscribers", func(t *testing.T) {
		cfg := defaultPublicDemoConfig()
		cfg.MaxSubscribers = 1
		_, base := newDemoTestServer(t, cfg)
		client := newDemoHTTPClient(t, base)
		provisionDemoSubscriber(t, client, "0000000031")
		resp, data := client.request(t, http.MethodPost, "/api/v1/subscribers", map[string]string{"imsi": "724000000000032", "msisdn": "+5500000000032"})
		requireDemoStatus(t, resp, data, http.StatusTooManyRequests)
	})

	t.Run("devices", func(t *testing.T) {
		cfg := defaultPublicDemoConfig()
		cfg.MaxDevices = 1
		_, base := newDemoTestServer(t, cfg)
		client := newDemoHTTPClient(t, base)
		sub := provisionDemoSubscriber(t, client, "0000000041")
		activateDemoSubscriber(t, client, sub.ID)
		registerDemoDevice(t, client, sub.ID, "0000000041")
		resp, data := client.request(t, http.MethodPost, "/api/v1/devices", map[string]string{"subscriber_id": sub.ID, "imei": "860000000000042", "technology": "5G"})
		requireDemoStatus(t, resp, data, http.StatusTooManyRequests)
	})

	t.Run("connected sessions", func(t *testing.T) {
		cfg := defaultPublicDemoConfig()
		cfg.MaxConnectedSessions = 1
		_, base := newDemoTestServer(t, cfg)
		client := newDemoHTTPClient(t, base)
		sub := provisionDemoSubscriber(t, client, "0000000051")
		activateDemoSubscriber(t, client, sub.ID)
		first := registerDemoDevice(t, client, sub.ID, "0000000051")
		second := registerDemoDevice(t, client, sub.ID, "0000000052")
		attachDemoSession(t, client, first.ID, "CELL-SP-001")
		resp, data := client.request(t, http.MethodPost, "/api/v1/sessions/attach", map[string]string{"device_id": second.ID, "cell_id": "CELL-SP-002"})
		requireDemoStatus(t, resp, data, http.StatusTooManyRequests)
	})
}

func TestPublicDemoMutationAndResetRateLimitsDoNotBlockReads(t *testing.T) {
	cfg := defaultPublicDemoConfig()
	cfg.MutationsPerWindow = 1
	cfg.ResetsPerWindow = 1
	_, base := newDemoTestServer(t, cfg)
	client := newDemoHTTPClient(t, base)
	provisionDemoSubscriber(t, client, "0000000061")
	resp, data := client.request(t, http.MethodPost, "/api/v1/subscribers", map[string]string{"imsi": "724000000000062", "msisdn": "+5500000000062"})
	requireDemoStatus(t, resp, data, http.StatusTooManyRequests)
	resp, data = client.request(t, http.MethodGet, "/api/v1/subscribers", nil)
	requireDemoStatus(t, resp, data, http.StatusOK)
	resp, data = client.request(t, http.MethodPost, "/api/v1/demo/reset", nil)
	requireDemoStatus(t, resp, data, http.StatusOK)
	resp, data = client.request(t, http.MethodPost, "/api/v1/demo/reset", nil)
	requireDemoStatus(t, resp, data, http.StatusTooManyRequests)
}

func TestPublicDemoCookieAndRequestSafety(t *testing.T) {
	cfg := defaultPublicDemoConfig()
	_, base := newDemoTestServer(t, cfg)

	for _, cookieValue := range []string{"", "invalid"} {
		req, err := http.NewRequest(http.MethodGet, base+"/api/v1/subscribers", nil)
		if err != nil {
			t.Fatal(err)
		}
		if cookieValue != "" {
			req.AddCookie(&http.Cookie{Name: publicDemoCookieName, Value: cookieValue})
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		var issued *http.Cookie
		for _, cookie := range resp.Cookies() {
			if cookie.Name == publicDemoCookieName {
				issued = cookie
			}
		}
		if issued == nil || !issued.HttpOnly || issued.SameSite != http.SameSiteStrictMode || issued.Path != "/" {
			t.Fatalf("invalid issued cookie: %+v", issued)
		}
		decoded, err := base64.RawURLEncoding.DecodeString(issued.Value)
		if err != nil || len(decoded) != 16 {
			t.Fatalf("cookie does not carry 128 random bits: len=%d err=%v", len(decoded), err)
		}
	}

	client := newDemoHTTPClient(t, base)
	req, _ := http.NewRequest(http.MethodPost, base+"/api/v1/subscribers", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "text/plain")
	resp, err := client.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("unsupported media type status=%d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodPost, base+"/api/v1/subscribers", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://attacker.example")
	resp, err = client.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-origin status=%d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodPost, base+"/api/v1/subscribers", strings.NewReader(strings.Repeat("x", int(cfg.RequestBodyLimit)+1)))
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body status=%d", resp.StatusCode)
	}
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" || resp.Header.Get("Content-Security-Policy") == "" || resp.Header.Get("X-Frame-Options") != "DENY" {
		t.Fatalf("security headers missing: %v", resp.Header)
	}
}

func TestPublicDemoConcurrentAccess(t *testing.T) {
	cfg := defaultPublicDemoConfig()
	cfg.MaxSubscribers = 50
	cfg.MutationsPerWindow = 100
	_, base := newDemoTestServer(t, cfg)
	clients := []*demoHTTPClient{newDemoHTTPClient(t, base), newDemoHTTPClient(t, base)}
	for _, client := range clients {
		resp, data := client.request(t, http.MethodGet, "/api/v1/subscribers", nil)
		requireDemoStatus(t, resp, data, http.StatusOK)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for clientIndex, client := range clients {
		for i := range 10 {
			wg.Add(1)
			go func(clientIndex, i int, client *demoHTTPClient) {
				defer wg.Done()
				suffix := fmt.Sprintf("%010d", 700+clientIndex*100+i)
				resp, data := client.request(t, http.MethodPost, "/api/v1/subscribers", map[string]string{"imsi": "72400" + suffix, "msisdn": "+55" + suffix})
				if resp.StatusCode != http.StatusCreated {
					errs <- fmt.Errorf("client=%d request=%d status=%d body=%s", clientIndex, i, resp.StatusCode, data)
				}
			}(clientIndex, i, client)
		}
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	for i, client := range clients {
		_, data := client.request(t, http.MethodGet, "/api/v1/subscribers", nil)
		if got := decodeDemoJSON[[]*subscriber.Subscriber](t, data); len(got) != 10 {
			t.Fatalf("visitor %d count=%d want=10", i, len(got))
		}
	}
}

func TestPublicDemoModeOffAndConfiguration(t *testing.T) {
	t.Setenv("PUBLIC_DEMO_MODE", "false")
	cfg, err := loadPublicDemoConfig()
	if err != nil || cfg.Enabled {
		t.Fatalf("mode should default to normal: cfg=%+v err=%v", cfg, err)
	}

	normal := httptest.NewServer(http.NewServeMux())
	defer normal.Close()
	resp, err := http.Post(normal.URL+"/api/v1/demo/reset", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("reset exists in normal mode: status=%d", resp.StatusCode)
	}

	t.Setenv("PUBLIC_DEMO_MODE", "true")
	t.Setenv("PUBLIC_DEMO_IDLE_TTL", "5m")
	t.Setenv("PUBLIC_DEMO_ABSOLUTE_TTL", "30m")
	cfg, err = loadPublicDemoConfig()
	if err != nil || !cfg.Enabled || cfg.IdleTTL != 5*time.Minute || cfg.AbsoluteTTL != 30*time.Minute {
		t.Fatalf("enabled config mismatch: cfg=%+v err=%v", cfg, err)
	}
}
