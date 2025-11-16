package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Thorlik/avito_internship/internal/app/dto"
	"github.com/Thorlik/avito_internship/internal/domain/models"
	"github.com/Thorlik/avito_internship/internal/domain/service"
)

type Handler struct {
	service *service.Service
}

func NewHandler(service *service.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var team models.Team
	if err := json.NewDecoder(r.Body).Decode(&team); err != nil {
		h.writeError(w, http.StatusBadRequest, models.ErrNotFound, "invalid request body")
		return
	}

	createdTeam, err := h.service.CreateTeam(r.Context(), &team)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, dto.TeamResponse{Team: *createdTeam})
}

func (h *Handler) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		h.writeError(w, http.StatusBadRequest, models.ErrNotFound, "team_name is required")
		return
	}

	team, err := h.service.GetTeam(r.Context(), teamName)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, team)
}

func (h *Handler) SetUserActive(w http.ResponseWriter, r *http.Request) {
	var req dto.SetUserActiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, models.ErrNotFound, "invalid request body")
		return
	}

	user, err := h.service.SetUserActive(r.Context(), req.UserID, req.IsActive)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, dto.UserResponse{User: *user})
}

func (h *Handler) GetUserReviews(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		h.writeError(w, http.StatusBadRequest, models.ErrNotFound, "user_id is required")
		return
	}

	prs, err := h.service.GetUserReviews(r.Context(), userID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, dto.UserReviewsResponse{
		UserID:            userID,
		PullRequestsShort: prs,
	})
}

func (h *Handler) CreatePullRequest(w http.ResponseWriter, r *http.Request) {
	var req dto.CreatePullRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, models.ErrNotFound, "invalid request body")
		return
	}

	pr, err := h.service.CreatePullRequest(r.Context(), req.PullRequestID, req.PullRequestName, req.AuthorID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, dto.PullRequestResponse{PR: *pr})
}

func (h *Handler) MergePullRequest(w http.ResponseWriter, r *http.Request) {
	var req dto.MergePullRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, models.ErrNotFound, "invalid request body")
		return
	}

	pr, err := h.service.MergePullRequest(r.Context(), req.PullRequestID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, dto.PullRequestResponse{PR: *pr})
}

func (h *Handler) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
	var req dto.ReassignReviewerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, models.ErrNotFound, "invalid request body")
		return
	}

	pr, newReviewerID, err := h.service.ReassignReviewer(r.Context(), req.PullRequestID, req.OldUserID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, dto.ReassignResponse{
		PR:         *pr,
		ReplacedBy: newReviewerID,
	})
}
