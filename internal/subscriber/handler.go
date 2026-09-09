package subscriber

import (
	"encoding/json"
	"errors"
	"net/http"
)

// Handler expõe as operações de domínio do Subscriber via HTTP REST.
type Handler struct {
	service *Service
}

// NewHandler instancia o handler com suas dependências de serviço.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registra os endpoints do domínio Subscriber no ServeMux fornecido.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/subscribers", h.handleProvision)
	mux.HandleFunc("GET /api/v1/subscribers/{id}", h.handleFindByID)
	mux.HandleFunc("GET /api/v1/subscribers", h.handleListOrFindByIMSI)
	mux.HandleFunc("POST /api/v1/subscribers/{id}/activate", h.handleActivate)
	mux.HandleFunc("POST /api/v1/subscribers/{id}/suspend", h.handleSuspend)
	mux.HandleFunc("POST /api/v1/subscribers/{id}/deactivate", h.handleDeactivate)
}

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

type actionReasonRequest struct {
	Reason string `json:"reason"`
}

func (h *Handler) handleProvision(w http.ResponseWriter, r *http.Request) {
	var req ProvisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "bad_request", "invalid request body", "INVALID_PAYLOAD")
		return
	}

	sub, err := h.service.Provision(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidIMSI), errors.Is(err, ErrInvalidMSISDN):
			writeJSONError(w, http.StatusBadRequest, "bad_request", err.Error(), "INVALID_TELECOM_IDENTITY")
		case errors.Is(err, ErrDuplicateIMSI), errors.Is(err, ErrDuplicateMSISDN):
			writeJSONError(w, http.StatusConflict, "conflict", err.Error(), "SUBSCRIBER_ALREADY_EXISTS")
		default:
			writeJSONError(w, http.StatusInternalServerError, "internal_error", "failed to provision subscriber", "INTERNAL_SERVER_ERROR")
		}
		return
	}

	writeJSON(w, http.StatusCreated, sub)
}

func (h *Handler) handleFindByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "bad_request", "subscriber id is required", "MISSING_IDENTIFIER")
		return
	}

	sub, err := h.service.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrSubscriberNotFound) {
			writeJSONError(w, http.StatusNotFound, "not_found", "subscriber not found", "SUBSCRIBER_NOT_FOUND")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "failed to query subscriber", "INTERNAL_SERVER_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, sub)
}

func (h *Handler) handleListOrFindByIMSI(w http.ResponseWriter, r *http.Request) {
	imsi := r.URL.Query().Get("imsi")
	if imsi != "" {
		sub, err := h.service.FindByIMSI(r.Context(), imsi)
		if err != nil {
			if errors.Is(err, ErrSubscriberNotFound) {
				writeJSONError(w, http.StatusNotFound, "not_found", "subscriber with given IMSI not found", "SUBSCRIBER_NOT_FOUND")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "internal_error", "failed to query subscriber by IMSI", "INTERNAL_SERVER_ERROR")
			return
		}
		writeJSON(w, http.StatusOK, sub)
		return
	}

	statusFilter := Status(r.URL.Query().Get("status"))
	subs, err := h.service.List(r.Context(), statusFilter)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "failed to list subscribers", "INTERNAL_SERVER_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, subs)
}

func (h *Handler) handleActivate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sub, err := h.service.Activate(r.Context(), id)
	if err != nil {
		handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, sub)
}

func (h *Handler) handleSuspend(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req actionReasonRequest
	if r.Body != nil && r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	sub, err := h.service.Suspend(r.Context(), id, req.Reason)
	if err != nil {
		handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, sub)
}

func (h *Handler) handleDeactivate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req actionReasonRequest
	if r.Body != nil && r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	sub, err := h.service.Deactivate(r.Context(), id, req.Reason)
	if err != nil {
		handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, sub)
}

func handleDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrSubscriberNotFound):
		writeJSONError(w, http.StatusNotFound, "not_found", "subscriber not found", "SUBSCRIBER_NOT_FOUND")
	case errors.Is(err, ErrAlreadyDeactivated):
		writeJSONError(w, http.StatusConflict, "conflict", err.Error(), "SUBSCRIBER_ALREADY_DEACTIVATED")
	case errors.Is(err, ErrInvalidTransition):
		writeJSONError(w, http.StatusConflict, "conflict", err.Error(), "INVALID_STATE_TRANSITION")
	default:
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "operation failed", "INTERNAL_SERVER_ERROR")
	}
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
