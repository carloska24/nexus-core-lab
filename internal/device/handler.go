package device

import (
	"encoding/json"
	"errors"
	"net/http"
)

// Handler expõe as operações do domínio Device via HTTP REST.
type Handler struct {
	service *Service
}

// NewHandler instancia o handler com suas dependências de serviço.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registra os endpoints do domínio Device no ServeMux fornecido.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/devices", h.handleRegister)
	mux.HandleFunc("GET /api/v1/devices/{id}", h.handleFindByID)
	mux.HandleFunc("GET /api/v1/devices", h.handleListBySubscriber)
}

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "bad_request", "invalid request body", "INVALID_PAYLOAD")
		return
	}

	dev, err := h.service.Register(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidIMEI), errors.Is(err, ErrInvalidTechnology), errors.Is(err, ErrMissingSubscriber):
			writeJSONError(w, http.StatusBadRequest, "bad_request", err.Error(), "INVALID_DEVICE_DATA")
		case errors.Is(err, ErrSubscriberNotFound):
			writeJSONError(w, http.StatusNotFound, "not_found", "associated subscriber not found", "SUBSCRIBER_NOT_FOUND")
		case errors.Is(err, ErrSubscriberNotActive):
			writeJSONError(w, http.StatusUnprocessableEntity, "unprocessable_entity", err.Error(), "SUBSCRIBER_NOT_ACTIVE")
		case errors.Is(err, ErrDuplicateIMEI):
			writeJSONError(w, http.StatusConflict, "conflict", err.Error(), "DEVICE_ALREADY_EXISTS")
		default:
			writeJSONError(w, http.StatusInternalServerError, "internal_error", "failed to register device", "INTERNAL_SERVER_ERROR")
		}
		return
	}

	writeJSON(w, http.StatusCreated, dev)
}

func (h *Handler) handleFindByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "bad_request", "device id is required", "MISSING_IDENTIFIER")
		return
	}

	dev, err := h.service.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrDeviceNotFound) {
			writeJSONError(w, http.StatusNotFound, "not_found", "device not found", "DEVICE_NOT_FOUND")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "failed to query device", "INTERNAL_SERVER_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, dev)
}

func (h *Handler) handleListBySubscriber(w http.ResponseWriter, r *http.Request) {
	subscriberID := r.URL.Query().Get("subscriber_id")
	if subscriberID == "" {
		writeJSONError(w, http.StatusBadRequest, "bad_request", "query parameter 'subscriber_id' is required", "MISSING_QUERY_PARAMETER")
		return
	}

	devices, err := h.service.ListBySubscriber(r.Context(), subscriberID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "failed to list devices", "INTERNAL_SERVER_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, devices)
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
