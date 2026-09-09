package subscriber

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupTestMux() (*http.ServeMux, *Service) {
	repo := NewMemoryRepository()
	service := NewService(repo)
	handler := NewHandler(service)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	return mux, service
}

func TestSubscriberHTTPHandlers(t *testing.T) {
	mux, _ := setupTestMux()

	var createdID string
	imsi := "724991112223334"
	msisdn := "+5519988776655"

	t.Run("POST /api/v1/subscribers - successful provisioning", func(t *testing.T) {
		body := `{"imsi": "` + imsi + `", "msisdn": "` + msisdn + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/subscribers", bytes.NewBufferString(body))
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
		}

		var res Subscriber
		if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if res.ID == "" || res.Status != StatusPendingActivation {
			t.Errorf("unexpected subscriber response: %+v", res)
		}
		createdID = res.ID
	})

	t.Run("POST /api/v1/subscribers - duplicate conflict 409", func(t *testing.T) {
		body := `{"imsi": "724991112223334", "msisdn": "+5519988776655"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/subscribers", bytes.NewBufferString(body))
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected status 409, got %d: %s", w.Code, w.Body.String())
		}

		var errRes errorResponse
		_ = json.NewDecoder(w.Body).Decode(&errRes)
		if errRes.Code != "SUBSCRIBER_ALREADY_EXISTS" {
			t.Errorf("expected code SUBSCRIBER_ALREADY_EXISTS, got %q", errRes.Code)
		}
	})

	t.Run("GET /api/v1/subscribers/{id} - found 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/subscribers/"+createdID, nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var res Subscriber
		_ = json.NewDecoder(w.Body).Decode(&res)
		if res.ID != createdID {
			t.Errorf("expected ID %s, got %s", createdID, res.ID)
		}
	})

	t.Run("GET /api/v1/subscribers?imsi=... - found 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/subscribers?imsi="+imsi, nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var res Subscriber
		_ = json.NewDecoder(w.Body).Decode(&res)
		if res.IMSI != imsi {
			t.Errorf("expected IMSI %s, got %s", imsi, res.IMSI)
		}
	})

	t.Run("POST /api/v1/subscribers/{id}/activate - 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/subscribers/"+createdID+"/activate", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var res Subscriber
		_ = json.NewDecoder(w.Body).Decode(&res)
		if res.Status != StatusActive {
			t.Errorf("expected status ACTIVE, got %s", res.Status)
		}
	})

	t.Run("POST /api/v1/subscribers/{id}/suspend - 200", func(t *testing.T) {
		body := `{"reason": "administrative_hold"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/subscribers/"+createdID+"/suspend", bytes.NewBufferString(body))
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var res Subscriber
		_ = json.NewDecoder(w.Body).Decode(&res)
		if res.Status != StatusSuspended || res.SuspensionReason != "administrative_hold" {
			t.Errorf("unexpected suspend state: %+v", res)
		}
	})

	t.Run("POST /api/v1/subscribers/{id}/deactivate - 200", func(t *testing.T) {
		body := `{"reason": "contract_ended"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/subscribers/"+createdID+"/deactivate", bytes.NewBufferString(body))
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var res Subscriber
		_ = json.NewDecoder(w.Body).Decode(&res)
		if res.Status != StatusDeactivated {
			t.Errorf("expected status DEACTIVATED, got %s", res.Status)
		}
	})

	t.Run("POST /api/v1/subscribers/{id}/activate - conflict 409 after DEACTIVATED", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/subscribers/"+createdID+"/activate", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected status 409, got %d: %s", w.Code, w.Body.String())
		}
	})
}
