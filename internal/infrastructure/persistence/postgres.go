package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Thorlik/avito_internship/internal/domain/models"
	_ "github.com/lib/pq"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(connectionString string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &PostgresStorage{db: db}, nil
}

func (s *PostgresStorage) Close() error {
	return s.db.Close()
}

func (s *PostgresStorage) CreateTeam(ctx context.Context, team *models.Team) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, "INSERT INTO teams (team_name) VALUES ($1)", team.TeamName)
	if err != nil {
		return err
	}

	for _, member := range team.Members {
		user := &models.User{
			UserID:   member.UserID,
			Username: member.Username,
			TeamName: team.TeamName,
			IsActive: member.IsActive,
		}
		err = s.upsertUserTx(ctx, tx, user)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *PostgresStorage) GetTeam(ctx context.Context, teamName string) (*models.Team, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)", teamName).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}

	rows, err := s.db.QueryContext(ctx,
		"SELECT user_id, username, is_active FROM users WHERE team_name = $1 ORDER BY user_id",
		teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := []models.TeamMember{}
	for rows.Next() {
		var member models.TeamMember
		if err := rows.Scan(&member.UserID, &member.Username, &member.IsActive); err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	return &models.Team{
		TeamName: teamName,
		Members:  members,
	}, nil
}

func (s *PostgresStorage) TeamExists(ctx context.Context, teamName string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)",
		teamName).Scan(&exists)
	return exists, err
}

func (s *PostgresStorage) CreateUser(ctx context.Context, user *models.User) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO users (user_id, username, team_name, is_active) VALUES ($1, $2, $3, $4)",
		user.UserID, user.Username, user.TeamName, user.IsActive)
	return err
}

func (s *PostgresStorage) UpdateUser(ctx context.Context, user *models.User) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE users SET username = $1, team_name = $2, is_active = $3 WHERE user_id = $4",
		user.Username, user.TeamName, user.IsActive, user.UserID)
	return err
}

func (s *PostgresStorage) GetUser(ctx context.Context, userID string) (*models.User, error) {
	user := &models.User{}
	err := s.db.QueryRowContext(ctx,
		"SELECT user_id, username, team_name, is_active FROM users WHERE user_id = $1",
		userID).Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *PostgresStorage) GetUsersByTeam(ctx context.Context, teamName string) ([]models.User, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT user_id, username, team_name, is_active FROM users WHERE team_name = $1",
		teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []models.User{}
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (s *PostgresStorage) upsertUserTx(ctx context.Context, tx *sql.Tx, user *models.User) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO users (user_id, username, team_name, is_active) 
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (user_id) 
		 DO UPDATE SET username = $2, team_name = $3, is_active = $4`,
		user.UserID, user.Username, user.TeamName, user.IsActive)
	return err
}

func (s *PostgresStorage) CreatePullRequest(ctx context.Context, pr *models.PullRequest) error {
	reviewersJSON, err := json.Marshal(pr.AssignedReviewers)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, assigned_reviewers, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		pr.PullRequestID, pr.PullRequestName, pr.AuthorID, pr.Status, reviewersJSON, pr.CreatedAt)
	return err
}

func (s *PostgresStorage) GetPullRequest(ctx context.Context, prID string) (*models.PullRequest, error) {
	var pr models.PullRequest
	var reviewersJSON []byte
	var createdAt, mergedAt sql.NullTime

	err := s.db.QueryRowContext(ctx,
		`SELECT pull_request_id, pull_request_name, author_id, status, assigned_reviewers, created_at, merged_at
		 FROM pull_requests WHERE pull_request_id = $1`,
		prID).Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status, &reviewersJSON, &createdAt, &mergedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(reviewersJSON, &pr.AssignedReviewers); err != nil {
		return nil, err
	}

	if createdAt.Valid {
		pr.CreatedAt = &createdAt.Time
	}
	if mergedAt.Valid {
		pr.MergedAt = &mergedAt.Time
	}

	return &pr, nil
}

func (s *PostgresStorage) UpdatePullRequest(ctx context.Context, pr *models.PullRequest) error {
	reviewersJSON, err := json.Marshal(pr.AssignedReviewers)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx,
		`UPDATE pull_requests 
		 SET pull_request_name = $1, author_id = $2, status = $3, assigned_reviewers = $4, merged_at = $5
		 WHERE pull_request_id = $6`,
		pr.PullRequestName, pr.AuthorID, pr.Status, reviewersJSON, pr.MergedAt, pr.PullRequestID)
	return err
}

func (s *PostgresStorage) PullRequestExists(ctx context.Context, prID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id = $1)",
		prID).Scan(&exists)
	return exists, err
}

func (s *PostgresStorage) GetPullRequestsByReviewer(ctx context.Context, userID string) ([]models.PullRequestShort, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT pull_request_id, pull_request_name, author_id, status
		 FROM pull_requests
		 WHERE assigned_reviewers::jsonb ? $1
		 ORDER BY created_at DESC`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prs := []models.PullRequestShort{}
	for rows.Next() {
		var pr models.PullRequestShort
		if err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status); err != nil {
			return nil, err
		}
		prs = append(prs, pr)
	}
	return prs, nil
}

func (s *PostgresStorage) GetReviewCounts(ctx context.Context, userIDs []string) (map[string]int, error) {
	if len(userIDs) == 0 {
		return map[string]int{}, nil
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT jsonb_array_elements_text(assigned_reviewers) as reviewer_id, COUNT(*) as count
		 FROM pull_requests
		 WHERE status = 'OPEN'
		 GROUP BY reviewer_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var userID string
		var count int
		if err := rows.Scan(&userID, &count); err != nil {
			return nil, err
		}
		counts[userID] = count
	}

	for _, userID := range userIDs {
		if _, exists := counts[userID]; !exists {
			counts[userID] = 0
		}
	}

	return counts, nil
}

func (s *PostgresStorage) GetStatistics(ctx context.Context) (*models.Statistics, error) {
	stats := &models.Statistics{}

	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM teams").Scan(&stats.TotalTeams)
	if err != nil {
		return nil, err
	}

	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*), COUNT(*) FILTER (WHERE is_active = true) FROM users").
		Scan(&stats.TotalUsers, &stats.ActiveUsers)
	if err != nil {
		return nil, err
	}

	err = s.db.QueryRowContext(ctx,
		`SELECT 
			COUNT(*), 
			COUNT(*) FILTER (WHERE status = 'OPEN'),
			COUNT(*) FILTER (WHERE status = 'MERGED')
		FROM pull_requests`).
		Scan(&stats.TotalPRs, &stats.OpenPRs, &stats.MergedPRs)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT 
			u.user_id,
			u.username,
			COUNT(*) FILTER (WHERE pr.status = 'OPEN') as open_reviews,
			COUNT(*) FILTER (WHERE pr.status = 'MERGED') as completed_reviews,
			COUNT(*) as total_reviews
		FROM users u
		LEFT JOIN pull_requests pr ON pr.assigned_reviewers::jsonb ? u.user_id
		GROUP BY u.user_id, u.username
		HAVING COUNT(*) > 0
		ORDER BY total_reviews DESC, open_reviews DESC
		LIMIT 10`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats.TopReviewers = []models.ReviewerStats{}
	for rows.Next() {
		var rs models.ReviewerStats
		if err := rows.Scan(&rs.UserID, &rs.Username, &rs.OpenReviews, &rs.CompletedReviews, &rs.TotalReviews); err != nil {
			return nil, err
		}
		stats.TopReviewers = append(stats.TopReviewers, rs)
	}

	return stats, nil
}
