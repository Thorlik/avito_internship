package service

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/Thorlik/avito_internship/internal/domain/models"
	"github.com/Thorlik/avito_internship/internal/domain/repository"
)

type Service struct {
	repo repository.Storage
	rng  *rand.Rand
}

func NewService(repo repository.Storage) *Service {
	return &Service{
		repo: repo,
		rng:  rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *Service) CreateTeam(ctx context.Context, team *models.Team) (*models.Team, error) {
	exists, err := s.repo.TeamExists(ctx, team.TeamName)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, &ServiceError{
			Code:    models.ErrTeamExists,
			Message: "team_name already exists",
		}
	}

	if err := s.repo.CreateTeam(ctx, team); err != nil {
		return nil, err
	}

	return s.repo.GetTeam(ctx, team.TeamName)
}

func (s *Service) GetTeam(ctx context.Context, teamName string) (*models.Team, error) {
	team, err := s.repo.GetTeam(ctx, teamName)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return nil, &ServiceError{
			Code:    models.ErrNotFound,
			Message: "team not found",
		}
	}
	return team, nil
}

func (s *Service) SetUserActive(ctx context.Context, userID string, isActive bool) (*models.User, error) {
	user, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, &ServiceError{
			Code:    models.ErrNotFound,
			Message: "user not found",
		}
	}

	user.IsActive = isActive
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) GetUserReviews(ctx context.Context, userID string) ([]models.PullRequestShort, error) {
	user, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return []models.PullRequestShort{}, nil
	}

	return s.repo.GetPullRequestsByReviewer(ctx, userID)
}

func (s *Service) CreatePullRequest(ctx context.Context, prID, prName, authorID string) (*models.PullRequest, error) {
	exists, err := s.repo.PullRequestExists(ctx, prID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, &ServiceError{
			Code:    models.ErrPRExists,
			Message: "PR id already exists",
		}
	}

	author, err := s.repo.GetUser(ctx, authorID)
	if err != nil {
		return nil, err
	}
	if author == nil {
		return nil, &ServiceError{
			Code:    models.ErrNotFound,
			Message: "author not found",
		}
	}

	teamMembers, err := s.repo.GetUsersByTeam(ctx, author.TeamName)
	if err != nil {
		return nil, err
	}

	reviewers := s.assignReviewers(teamMembers, authorID)

	now := time.Now()
	pr := &models.PullRequest{
		PullRequestID:     prID,
		PullRequestName:   prName,
		AuthorID:          authorID,
		Status:            models.StatusOpen,
		AssignedReviewers: reviewers,
		CreatedAt:         &now,
	}

	if err := s.repo.CreatePullRequest(ctx, pr); err != nil {
		return nil, err
	}

	return pr, nil
}

func (s *Service) MergePullRequest(ctx context.Context, prID string) (*models.PullRequest, error) {
	pr, err := s.repo.GetPullRequest(ctx, prID)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, &ServiceError{
			Code:    models.ErrNotFound,
			Message: "PR not found",
		}
	}

	if pr.Status == models.StatusMerged {
		return pr, nil
	}

	now := time.Now()
	pr.Status = models.StatusMerged
	pr.MergedAt = &now

	if err := s.repo.UpdatePullRequest(ctx, pr); err != nil {
		return nil, err
	}

	return pr, nil
}

func (s *Service) ReassignReviewer(ctx context.Context, prID, oldReviewerID string) (*models.PullRequest, string, error) {
	pr, err := s.repo.GetPullRequest(ctx, prID)
	if err != nil {
		return nil, "", err
	}
	if pr == nil {
		return nil, "", &ServiceError{
			Code:    models.ErrNotFound,
			Message: "PR not found",
		}
	}

	if pr.Status == models.StatusMerged {
		return nil, "", &ServiceError{
			Code:    models.ErrPRMerged,
			Message: "cannot reassign on merged PR",
		}
	}

	reviewerIndex := -1
	for i, reviewerID := range pr.AssignedReviewers {
		if reviewerID == oldReviewerID {
			reviewerIndex = i
			break
		}
	}
	if reviewerIndex == -1 {
		return nil, "", &ServiceError{
			Code:    models.ErrNotAssigned,
			Message: "reviewer is not assigned to this PR",
		}
	}

	oldReviewer, err := s.repo.GetUser(ctx, oldReviewerID)
	if err != nil {
		return nil, "", err
	}
	if oldReviewer == nil {
		return nil, "", &ServiceError{
			Code:    models.ErrNotFound,
			Message: "old reviewer not found",
		}
	}

	teamMembers, err := s.repo.GetUsersByTeam(ctx, oldReviewer.TeamName)
	if err != nil {
		return nil, "", err
	}

	newReviewerID, err := s.findReplacement(ctx, teamMembers, pr.AuthorID, pr.AssignedReviewers)
	if err != nil {
		return nil, "", err
	}

	pr.AssignedReviewers[reviewerIndex] = newReviewerID
	if err := s.repo.UpdatePullRequest(ctx, pr); err != nil {
		return nil, "", err
	}

	return pr, newReviewerID, nil
}

func (s *Service) assignReviewers(teamMembers []models.User, authorID string) []string {
	candidates := []models.User{}
	candidateIDs := []string{}
	for _, member := range teamMembers {
		if member.IsActive && member.UserID != authorID {
			candidates = append(candidates, member)
			candidateIDs = append(candidateIDs, member.UserID)
		}
	}

	if len(candidates) == 0 {
		return []string{}
	}

	counts, err := s.repo.GetReviewCounts(context.Background(), candidateIDs)
	if err != nil {
		return s.randomSelection(candidates, 2)
	}

	sort.Slice(candidates, func(i, j int) bool {
		countI := counts[candidates[i].UserID]
		countJ := counts[candidates[j].UserID]
		if countI != countJ {
			return countI < countJ
		}
		return candidates[i].UserID < candidates[j].UserID
	})

	reviewers := []string{}
	for i := 0; i < len(candidates) && i < 2; i++ {
		reviewers = append(reviewers, candidates[i].UserID)
	}

	return reviewers
}

func (s *Service) findReplacement(ctx context.Context, teamMembers []models.User, authorID string, currentReviewers []string) (string, error) {
	excluded := make(map[string]bool)
	excluded[authorID] = true
	for _, reviewerID := range currentReviewers {
		excluded[reviewerID] = true
	}

	candidates := []models.User{}
	for _, member := range teamMembers {
		if member.IsActive && !excluded[member.UserID] {
			candidates = append(candidates, member)
		}
	}

	if len(candidates) == 0 {
		return "", &ServiceError{
			Code:    models.ErrNoCandidate,
			Message: "no active replacement candidate in team",
		}
	}

	candidateIDs := []string{}
	for _, c := range candidates {
		candidateIDs = append(candidateIDs, c.UserID)
	}
	counts, err := s.repo.GetReviewCounts(ctx, candidateIDs)
	if err != nil {
		return candidates[s.rng.Intn(len(candidates))].UserID, nil
	}

	minCount := -1
	var selected []models.User
	for _, candidate := range candidates {
		count := counts[candidate.UserID]
		if minCount == -1 || count < minCount {
			minCount = count
			selected = []models.User{candidate}
		} else if count == minCount {
			selected = append(selected, candidate)
		}
	}

	if len(selected) > 0 {
		return selected[s.rng.Intn(len(selected))].UserID, nil
	}

	return "", &ServiceError{
		Code:    models.ErrNoCandidate,
		Message: "no active replacement candidate in team",
	}
}

func (s *Service) randomSelection(candidates []models.User, maxCount int) []string {
	if len(candidates) <= maxCount {
		result := make([]string, len(candidates))
		for i, c := range candidates {
			result[i] = c.UserID
		}
		return result
	}

	s.rng.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	result := make([]string, maxCount)
	for i := 0; i < maxCount; i++ {
		result[i] = candidates[i].UserID
	}
	return result
}

type ServiceError struct {
	Code    models.ErrorCode
	Message string
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
