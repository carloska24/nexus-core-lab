package session

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/carloska24/nexus-core-lab/internal/network"
)

// Handler expõe as operações do domínio Session via HTTP REST.
type Handler struct {
	service *Service
}

// NewHandler instancia o handler de sessões com suas dependências de serviço.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registra os endpoints do domínio Session no ServeMux fornecido.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/sessions/attach", h.handleAttach)
	mux.HandleFunc("POST /api/v1/sessions/{id}/handover", h.handleHandover)
	mux.HandleFunc("POST /api/v1/sessions/{id}/detach", h.handleDetach)
	mux.HandleFunc("GET /api/v1/sessions/{id}", h.handleFindByID)
	mux.HandleFunc("GET /api/v1/sessions", h.handleGetActiveByDevice)
}

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

func (h *Handler) handleAttach(w http.ResponseWriter, r *http.Request) {
	var req AttachRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "bad_request", "invalid request body", "INVALID_PAYLOAD")
		return
	}

	session, err := h.service.Attach(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrMissingDeviceID), errors.Is(err, ErrMissingCellID):
			writeJSONError(w, http.StatusBadRequest, "bad_request", err.Error(), "MISSING_ATTACH_FIELD")
		case errors.Is(err, network.ErrCellNotFound):
			writeJSONError(w, http.StatusBadRequest, "bad_request", "specified cell does not exist in network topology", "CELL_NOT_FOUND")
		case errors.Is(err, ErrDeviceNotFound):
			writeJSONError(w, http.StatusNotFound, "not_found", "associated device not found", "DEVICE_NOT_FOUND")
		case errors.Is(err, ErrDeviceNotEligible):
			writeJSONError(w, http.StatusUnprocessableEntity, "unprocessable_entity", "device is not eligible for network attach", "DEVICE_NOT_ELIGIBLE")
		default:
			writeJSONError(w, http.StatusInternalServerError, "internal_error", "failed to attach session", "INTERNAL_SERVER_ERROR")
		}
		return
	}

	writeJSON(w, http.StatusCreated, session)
}

func (h *Handler) handleHandover(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "bad_request", "session id is required", "MISSING_IDENTIFIER")
		return
	}

	var req HandoverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "bad_request", "invalid request body", "INVALID_PAYLOAD")
		return
	}

	session, err := h.service.Handover(r.Context(), id, req.TargetCellID)
	if err != nil {
		switch {
		case errors.Is(err, ErrSessionNotFound):
			writeJSONError(w, http.StatusNotFound, "not_found", "session not found", "SESSION_NOT_FOUND")
		case errors.Is(err, ErrSessionNotConnected):
			writeJSONError(w, http.StatusUnprocessableEntity, "unprocessable_entity", "session is not connected", "SESSION_NOT_CONNECTED")
		case errors.Is(err, network.ErrCellNotFound):
			writeJSONError(w, http.StatusBadRequest, "bad_request", "target cell does not exist in network topology", "TARGET_CELL_NOT_FOUND")
		default:
			writeJSONError(w, http.StatusInternalServerError, "internal_error", "failed to perform handover", "INTERNAL_SERVER_ERROR")
		}
		return
	}

	writeJSON(w, http.StatusOK, session)
}

func (h *Handler) handleDetach(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "bad_request", "session id is required", "MISSING_IDENTIFIER")
		return
	}

	session, err := h.service.Detach(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			writeJSONError(w, http.StatusNotFound, "not_found", "session not found", "SESSION_NOT_FOUND")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "failed to detach session", "INTERNAL_SERVER_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, session)
}

func (h *Handler) handleFindByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "bad_request", "session id is required", "MISSING_IDENTIFIER")
		return
	}

	session, err := h.service.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			writeJSONError(w, http.StatusNotFound, "not_found", "session not found", "SESSION_NOT_FOUND")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "failed to query session", "INTERNAL_SERVER_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, session)
}

func (h *Handler) handleGetActiveByDevice(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		writeJSONError(w, http.StatusBadRequest, "bad_request", "query parameter 'device_id' is required", "MISSING_QUERY_PARAMETER")
		return
	}

	session, err := h.service.GetActiveByDevice(r.Context(), deviceID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			writeJSONError(w, http.StatusNotFound, "not_found", "no active session found for device", "ACTIVE_SESSION_NOT_FOUND")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "failed to query active session", "INTERNAL_SERVER_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, session)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, errStr, msg, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{
		Error:   errStr,
		Message: msg,
		Code:    code,
	})
}
