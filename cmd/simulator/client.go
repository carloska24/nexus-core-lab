package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DTOs locais para consumo agnóstico da API externa.
// Nenhum tipo ou pacote de domínio interno é importado.

type ProvisionRequest struct {
	IMSI   string `json:"imsi"`
	MSISDN string `json:"msisdn"`
}

type ProvisionResponse struct {
	ID     string `json:"id"`
	IMSI   string `json:"imsi"`
	MSISDN string `json:"msisdn"`
	Status string `json:"status"`
}

type DeviceRegisterRequest struct {
	SubscriberID string `json:"subscriber_id"`
	IMEI         string `json:"imei"`
	Technology   string `json:"technology"`
}

type DeviceResponse struct {
	ID           string `json:"id"`
	SubscriberID string `json:"subscriber_id"`
	IMEI         string `json:"imei"`
	Technology   string `json:"technology"`
	Status       string `json:"status"`
}

type AttachRequest struct {
	DeviceID string `json:"device_id"`
	CellID   string `json:"cell_id"`
}

type AttachResponse struct {
	ID        string `json:"id"`
	DeviceID  string `json:"device_id"`
	CellID    string `json:"cell_id"`
	IPAddress string `json:"ip_address"`
	Status    string `json:"status"`
}

type HandoverRequest struct {
	TargetCellID string `json:"target_cell_id"`
}

type SessionResponse struct {
	ID        string `json:"id"`
	DeviceID  string `json:"device_id"`
	CellID    string `json:"cell_id"`
	IPAddress string `json:"ip_address"`
	Status    string `json:"status"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

type TelemetryMetrics struct {
	ActiveSessions     int64  `json:"active_sessions"`
	ConnectedDevices   int64  `json:"connected_devices"`
	DroppedEventsTotal uint64 `json:"dropped_events_total"`
}

type TelemetryEvents struct {
	Attach          uint64 `json:"attach"`
	CellHandover    uint64 `json:"cell_handover"`
	Detach          uint64 `json:"detach"`
	StaleDisconnect uint64 `json:"stale_disconnect"`
}

type TelemetrySnapshot struct {
	Service       string           `json:"service"`
	Timestamp     time.Time        `json:"timestamp"`
	RequestsTotal uint64           `json:"requests_total"`
	Metrics       TelemetryMetrics `json:"metrics"`
	EventsTotal   TelemetryEvents  `json:"events_total"`
}

// Client provê acesso HTTP direto aos endpoints REST do monólito.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient instancia um cliente com timeout explícito e pool de conexões reutilizáveis.
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     30 * time.Second,
			},
		},
	}
}

func (c *Client) doRequest(ctx context.Context, method, path string, body any, expectedStatus int, out any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != expectedStatus {
		var errResp ErrorResponse
		if jsonErr := json.Unmarshal(respBody, &errResp); jsonErr == nil && errResp.Message != "" {
			return fmt.Errorf("HTTP %d: %s (%s)", resp.StatusCode, errResp.Message, errResp.Code)
		}
		return fmt.Errorf("HTTP %d: unexpected status code (expected %d): %s", resp.StatusCode, expectedStatus, string(respBody))
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("failed to unmarshal response json: %w", err)
		}
	}

	return nil
}

// ProvisionSubscriber executa POST /api/v1/subscribers.
func (c *Client) ProvisionSubscriber(ctx context.Context, imsi, msisdn string) (*ProvisionResponse, error) {
	req := ProvisionRequest{
		IMSI:   imsi,
		MSISDN: msisdn,
	}
	var res ProvisionResponse
	if err := c.doRequest(ctx, http.MethodPost, "/api/v1/subscribers", req, http.StatusCreated, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// ActivateSubscriber executa POST /api/v1/subscribers/{id}/activate.
func (c *Client) ActivateSubscriber(ctx context.Context, subscriberID string) error {
	path := fmt.Sprintf("/api/v1/subscribers/%s/activate", subscriberID)
	return c.doRequest(ctx, http.MethodPost, path, nil, http.StatusOK, nil)
}

// RegisterDevice executa POST /api/v1/devices.
func (c *Client) RegisterDevice(ctx context.Context, subscriberID, imei, tech string) (*DeviceResponse, error) {
	req := DeviceRegisterRequest{
		SubscriberID: subscriberID,
		IMEI:         imei,
		Technology:   tech,
	}
	var res DeviceResponse
	if err := c.doRequest(ctx, http.MethodPost, "/api/v1/devices", req, http.StatusCreated, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// AttachSession executa POST /api/v1/sessions/attach.
func (c *Client) AttachSession(ctx context.Context, deviceID, cellID string) (*AttachResponse, error) {
	req := AttachRequest{
		DeviceID: deviceID,
		CellID:   cellID,
	}
	var res AttachResponse
	if err := c.doRequest(ctx, http.MethodPost, "/api/v1/sessions/attach", req, http.StatusCreated, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// HandoverSession executa POST /api/v1/sessions/{id}/handover.
func (c *Client) HandoverSession(ctx context.Context, sessionID, targetCellID string) error {
	path := fmt.Sprintf("/api/v1/sessions/%s/handover", sessionID)
	req := HandoverRequest{
		TargetCellID: targetCellID,
	}
	var res SessionResponse
	return c.doRequest(ctx, http.MethodPost, path, req, http.StatusOK, &res)
}

// DetachSession executa POST /api/v1/sessions/{id}/detach.
func (c *Client) DetachSession(ctx context.Context, sessionID string) error {
	path := fmt.Sprintf("/api/v1/sessions/%s/detach", sessionID)
	var res SessionResponse
	return c.doRequest(ctx, http.MethodPost, path, nil, http.StatusOK, &res)
}

// GetTelemetry executa GET /telemetry.
func (c *Client) GetTelemetry(ctx context.Context) (*TelemetrySnapshot, error) {
	var res TelemetrySnapshot
	if err := c.doRequest(ctx, http.MethodGet, "/telemetry", nil, http.StatusOK, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
