package device

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupTestDeviceMux() (*http.ServeMux, *mockSubscriberChecker) {
	checker := &mockSubscriberChecker{
		subscribers: map[string]string{
			"sub-active":    "ACTIVE",
			"sub-suspended": "SUSPENDED",
		},
	}

	repo := NewMemoryRepository()
	service := NewService(repo, checker)
	handler := NewHandler(service)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	return mux, checker
}

func TestDeviceHTTPHandlers(t *testing.T) {
	mux, _ := setupTestDeviceMux()

	var createdID string
	imei := "356938035643810"

	t.Run("POST /api/v1/devices - successful registration 201", func(t *testing.T) {
		body := `{"subscriber_id": "sub-active", "imei": "` + imei + `", "technology": "5G"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewBufferString(body))
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
		}

		var res Device
		if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if res.ID == "" || res.Status != StatusRegistered || res.Technology != Tech5G {
			t.Errorf("unexpected device response: %+v", res)
		}
		createdID = res.ID
	})

	t.Run("POST /api/v1/devices - duplicate IMEI 409", func(t *testing.T) {
		body := `{"subscriber_id": "sub-active", "imei": "` + imei + `", "technology": "LTE"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewBufferString(body))
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected status 409, got %d: %s", w.Code, w.Body.String())
		}

		var errRes errorResponse
		_ = json.NewDecoder(w.Body).Decode(&errRes)
		if errRes.Code != "DEVICE_ALREADY_EXISTS" {
			t.Errorf("expected code DEVICE_ALREADY_EXISTS, got %q", errRes.Code)
		}
	})

	t.Run("POST /api/v1/devices - non-existent subscriber 404", func(t *testing.T) {
		body := `{"subscriber_id": "unknown-sub", "imei": "356938035643899", "technology": "LTE"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewBufferString(body))
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("POST /api/v1/devices - non-active subscriber 422", func(t *testing.T) {
		body := `{"subscriber_id": "sub-suspended", "imei": "356938035643899", "technology": "LTE"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewBufferString(body))
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected status 422, got %d: %s", w.Code, w.Body.String())
		}

		var errRes errorResponse
		_ = json.NewDecoder(w.Body).Decode(&errRes)
		if errRes.Code != "SUBSCRIBER_NOT_ACTIVE" {
			t.Errorf("expected code SUBSCRIBER_NOT_ACTIVE, got %q", errRes.Code)
		}
	})

	t.Run("POST /api/v1/devices - invalid IMEI 400", func(t *testing.T) {
		body := `{"subscriber_id": "sub-active", "imei": "12345", "technology": "5G"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewBufferString(body))
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("GET /api/v1/devices/{id} - found 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/"+createdID, nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var res Device
		_ = json.NewDecoder(w.Body).Decode(&res)
		if res.ID != createdID {
			t.Errorf("expected ID %s, got %s", createdID, res.ID)
		}
	})

	t.Run("GET /api/v1/devices/{id} - not found 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/non-existent-id", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("GET /api/v1/devices?subscriber_id=... - 200 list", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/devices?subscriber_id=sub-active", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var list []Device
		_ = json.NewDecoder(w.Body).Decode(&list)
		if len(list) != 1 || list[0].ID != createdID {
			t.Errorf("unexpected list result: %+v", list)
		}
	})

	t.Run("GET /api/v1/devices without subscriber_id query - 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
		}
	})
}
