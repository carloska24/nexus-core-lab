package session

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/carloska24/nexus-core-lab/internal/network"
)

func setupTestServer() (http.Handler, *mockDeviceChecker) {
	repo := NewMemoryRepository()
	checker := newMockDeviceChecker()
	ipPool := network.NewIPPool()
	svc := NewService(repo, checker, ipPool)
	h := NewHandler(svc)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux, checker
}

func TestHTTP_Attach(t *testing.T) {
	mux, checker := setupTestServer()
	checker.addDevice("DEV-HTTP-001", "SUB-HTTP-001", true)
	checker.addDevice("DEV-HTTP-INELIGIBLE", "SUB-HTTP-002", false)

	// 1. Sucesso
	body, _ := json.Marshal(AttachRequest{
		DeviceID: "DEV-HTTP-001",
		CellID:   "CELL-SP-001",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/attach", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var created Session
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if created.Status != StatusConnected {
		t.Errorf("expected StatusConnected, got %s", created.Status)
	}

	// 2. Dispositivo não elegível -> 422
	bodyInel, _ := json.Marshal(AttachRequest{
		DeviceID: "DEV-HTTP-INELIGIBLE",
		CellID:   "CELL-SP-001",
	})
	reqInel := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/attach", bytes.NewReader(bodyInel))
	wInel := httptest.NewRecorder()
	mux.ServeHTTP(wInel, reqInel)
	if wInel.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", wInel.Code)
	}

	// 3. Dispositivo inexistente -> 404
	bodyUnknown, _ := json.Marshal(AttachRequest{
		DeviceID: "DEV-NOT-FOUND",
		CellID:   "CELL-SP-001",
	})
	reqUnknown := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/attach", bytes.NewReader(bodyUnknown))
	wUnknown := httptest.NewRecorder()
	mux.ServeHTTP(wUnknown, reqUnknown)
	if wUnknown.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", wUnknown.Code)
	}

	// 4. Célula inválida -> 400
	bodyBadCell, _ := json.Marshal(AttachRequest{
		DeviceID: "DEV-HTTP-001",
		CellID:   "CELL-INVALID",
	})
	reqBadCell := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/attach", bytes.NewReader(bodyBadCell))
	wBadCell := httptest.NewRecorder()
	mux.ServeHTTP(wBadCell, reqBadCell)
	if wBadCell.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", wBadCell.Code)
	}
}

func TestHTTP_HandoverAndDetach(t *testing.T) {
	mux, checker := setupTestServer()
	checker.addDevice("DEV-001", "SUB-001", true)

	// Anexa sessão
	bodyAttach, _ := json.Marshal(AttachRequest{DeviceID: "DEV-001", CellID: "CELL-SP-001"})
	reqAttach := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/attach", bytes.NewReader(bodyAttach))
	wAttach := httptest.NewRecorder()
	mux.ServeHTTP(wAttach, reqAttach)

	var sess Session
	_ = json.NewDecoder(wAttach.Body).Decode(&sess)

	// Handover para CELL-SP-002
	bodyHO, _ := json.Marshal(HandoverRequest{TargetCellID: "CELL-SP-002"})
	reqHO := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+sess.ID+"/handover", bytes.NewReader(bodyHO))
	wHO := httptest.NewRecorder()
	mux.ServeHTTP(wHO, reqHO)

	if wHO.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on handover, got %d: %s", wHO.Code, wHO.Body.String())
	}

	var hoSess Session
	_ = json.NewDecoder(wHO.Body).Decode(&hoSess)
	if hoSess.CellID != "CELL-SP-002" {
		t.Errorf("expected CELL-SP-002, got %s", hoSess.CellID)
	}

	// Detach
	reqDetach := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+sess.ID+"/detach", nil)
	wDetach := httptest.NewRecorder()
	mux.ServeHTTP(wDetach, reqDetach)

	if wDetach.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on detach, got %d", wDetach.Code)
	}

	// Handover posterior em sessão desconectada -> 422
	reqBadHO := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+sess.ID+"/handover", bytes.NewReader(bodyHO))
	wBadHO := httptest.NewRecorder()
	mux.ServeHTTP(wBadHO, reqBadHO)

	if wBadHO.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 on handover after detach, got %d", wBadHO.Code)
	}

	// Detach idempotente -> 200 OK
	reqDetach2 := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+sess.ID+"/detach", nil)
	wDetach2 := httptest.NewRecorder()
	mux.ServeHTTP(wDetach2, reqDetach2)
	if wDetach2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on idempotent detach, got %d", wDetach2.Code)
	}
}

func TestHTTP_FindByIDAndGetActive(t *testing.T) {
	mux, checker := setupTestServer()
	checker.addDevice("DEV-001", "SUB-001", true)

	// Anexa
	bodyAttach, _ := json.Marshal(AttachRequest{DeviceID: "DEV-001", CellID: "CELL-SP-001"})
	reqAttach := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/attach", bytes.NewReader(bodyAttach))
	wAttach := httptest.NewRecorder()
	mux.ServeHTTP(wAttach, reqAttach)

	var sess Session
	_ = json.NewDecoder(wAttach.Body).Decode(&sess)

	// Find by ID
	reqFind := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/"+sess.ID, nil)
	wFind := httptest.NewRecorder()
	mux.ServeHTTP(wFind, reqFind)
	if wFind.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", wFind.Code)
	}

	// Get Active by device_id
	reqActive := httptest.NewRequest(http.MethodGet, "/api/v1/sessions?device_id=DEV-001", nil)
	wActive := httptest.NewRecorder()
	mux.ServeHTTP(wActive, reqActive)
	if wActive.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", wActive.Code)
	}

	// Query sem device_id -> 400
	reqNoParam := httptest.NewRequest(http.MethodGet, "/api/v1/sessions", nil)
	wNoParam := httptest.NewRecorder()
	mux.ServeHTTP(wNoParam, reqNoParam)
	if wNoParam.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", wNoParam.Code)
	}
}
