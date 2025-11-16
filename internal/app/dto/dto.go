package dto

import (
	"github.com/Thorlik/avito_internship/internal/domain/models"
)

type CreatePullRequestRequest struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
}

type MergePullRequestRequest struct {
	PullRequestID string `json:"pull_request_id"`
}

type ReassignReviewerRequest struct {
	PullRequestID string `json:"pull_request_id"`
	OldUserID     string `json:"old_user_id"`
}

type SetUserActiveRequest struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

type TeamResponse struct {
	Team models.Team `json:"team"`
}

type UserResponse struct {
	User models.User `json:"user"`
}

type PullRequestResponse struct {
	PR models.PullRequest `json:"pr"`
}

type ReassignResponse struct {
	PR         models.PullRequest `json:"pr"`
	ReplacedBy string             `json:"replaced_by"`
}

type UserReviewsResponse struct {
	UserID            string                    `json:"user_id"`
	PullRequestsShort []models.PullRequestShort `json:"pull_requests"`
}
