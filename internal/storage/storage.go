package storage

import (
	"context"

	"github.com/Thorlik/avito_internship/internal/models"
)

type Storage interface {
	CreateTeam(ctx context.Context, team *models.Team) error
	GetTeam(ctx context.Context, teamName string) (*models.Team, error)
	TeamExists(ctx context.Context, teamName string) (bool, error)

	CreateUser(ctx context.Context, user *models.User) error
	UpdateUser(ctx context.Context, user *models.User) error
	GetUser(ctx context.Context, userID string) (*models.User, error)
	GetUsersByTeam(ctx context.Context, teamName string) ([]models.User, error)

	CreatePullRequest(ctx context.Context, pr *models.PullRequest) error
	GetPullRequest(ctx context.Context, prID string) (*models.PullRequest, error)
	UpdatePullRequest(ctx context.Context, pr *models.PullRequest) error
	PullRequestExists(ctx context.Context, prID string) (bool, error)
	GetPullRequestsByReviewer(ctx context.Context, userID string) ([]models.PullRequestShort, error)

	GetReviewCounts(ctx context.Context, userIDs []string) (map[string]int, error)

	Close() error
}
