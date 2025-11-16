package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Thorlik/avito_internship/internal/domain/models"
	"github.com/Thorlik/avito_internship/internal/domain/service"
)

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, code models.ErrorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(models.ErrorResponse{
		Error: models.ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
	if serviceErr, ok := err.(*service.ServiceError); ok {
		status := http.StatusInternalServerError
		switch serviceErr.Code {
		case models.ErrTeamExists:
			status = http.StatusBadRequest
		case models.ErrPRExists, models.ErrPRMerged, models.ErrNotAssigned, models.ErrNoCandidate:
			status = http.StatusConflict
		case models.ErrNotFound:
			status = http.StatusNotFound
		}
		h.writeError(w, status, serviceErr.Code, serviceErr.Message)
		return
	}
	h.writeError(w, http.StatusInternalServerError, models.ErrNotFound, "internal server error")
}
