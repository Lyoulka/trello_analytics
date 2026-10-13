package handlers

import (
	"encoding/json"
	"net/http"
)

// APIError is the JSON error envelope (Architect Decision).
type APIError struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody is the inner error object.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, APIError{Error: ErrorBody{Code: code, Message: message}})
}

// WriteUnauthorized writes a 401 error envelope.
func WriteUnauthorized(w http.ResponseWriter) {
	writeError(w, http.StatusUnauthorized, "unauthenticated", "authentication required")
}

// WriteForbidden writes a 403 error envelope.
func WriteForbidden(w http.ResponseWriter) {
	writeError(w, http.StatusForbidden, "forbidden", "insufficient permissions")
}

func writeBadRequest(w http.ResponseWriter, code, message string) {
	writeError(w, http.StatusBadRequest, code, message)
}

func writeNotFound(w http.ResponseWriter, message string) {
	writeError(w, http.StatusNotFound, "not_found", message)
}

// WriteInternal writes a 500 error envelope without leaking internals.
func WriteInternal(w http.ResponseWriter) {
	writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
}
