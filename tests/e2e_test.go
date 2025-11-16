package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

const baseURL = "http://localhost:8080"

type TestClient struct {
	baseURL string
	client  *http.Client
}

func NewTestClient() *TestClient {
	return &TestClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *TestClient) request(method, endpoint string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, c.baseURL+endpoint, reqBody)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.client.Do(req)
}

func (c *TestClient) post(endpoint string, body interface{}) (*http.Response, error) {
	return c.request("POST", endpoint, body)
}

func (c *TestClient) get(endpoint string) (*http.Response, error) {
	return c.request("GET", endpoint, nil)
}

func TestE2E_FullWorkflow(t *testing.T) {
	client := NewTestClient()

	resp, err := client.get("/statistics")
	if err != nil {
		t.Fatalf("Service is not available: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Service returned unexpected status: %d", resp.StatusCode)
	}

	t.Run("CreateTeam", func(t *testing.T) {
		team := map[string]interface{}{
			"team_name": "e2e_team_" + fmt.Sprint(time.Now().Unix()),
			"members": []map[string]interface{}{
				{"user_id": "e2e_user1", "username": "Alice", "is_active": true},
				{"user_id": "e2e_user2", "username": "Bob", "is_active": true},
				{"user_id": "e2e_user3", "username": "Charlie", "is_active": true},
				{"user_id": "e2e_user4", "username": "David", "is_active": true},
			},
		}

		resp, err := client.post("/team/add", team)
		if err != nil {
			t.Fatalf("Failed to create team: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected status 201, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var result struct {
			Team struct {
				TeamName string `json:"team_name"`
			} `json:"team"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result.Team.TeamName != team["team_name"] {
			t.Errorf("Expected team_name %s, got %s", team["team_name"], result.Team.TeamName)
		}
	})

	t.Run("GetTeam", func(t *testing.T) {
		resp, err := client.get("/team/get?team_name=e2e_team_" + fmt.Sprint(time.Now().Unix()))
		if err != nil {
			t.Fatalf("Failed to get team: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
			t.Fatalf("Expected status 200 or 404, got %d", resp.StatusCode)
		}
	})

	t.Run("SetUserActive", func(t *testing.T) {
		payload := map[string]interface{}{
			"user_id":   "e2e_user1",
			"is_active": false,
		}

		resp, err := client.post("/users/setIsActive", payload)
		if err != nil {
			t.Fatalf("Failed to set user active: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})

	var prID string
	t.Run("CreatePullRequest", func(t *testing.T) {
		prID = "e2e_pr_" + fmt.Sprint(time.Now().Unix())
		pr := map[string]interface{}{
			"pull_request_id":   prID,
			"pull_request_name": "E2E Test PR",
			"author_id":         "e2e_user1",
		}

		resp, err := client.post("/pullRequest/create", pr)
		if err != nil {
			t.Fatalf("Failed to create PR: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected status 201, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var result struct {
			PR struct {
				PullRequestID     string   `json:"pull_request_id"`
				AssignedReviewers []string `json:"assigned_reviewers"`
			} `json:"pr"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result.PR.PullRequestID != prID {
			t.Errorf("Expected PR ID %s, got %s", prID, result.PR.PullRequestID)
		}

		if len(result.PR.AssignedReviewers) == 0 {
			t.Error("Expected reviewers to be assigned")
		}
	})

	t.Run("GetUserReviews", func(t *testing.T) {
		resp, err := client.get("/users/getReview?user_id=e2e_user2")
		if err != nil {
			t.Fatalf("Failed to get user reviews: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if _, ok := result["pull_requests"]; !ok {
			t.Error("Expected pull_requests field in response")
		}
	})

	t.Run("ReassignReviewer", func(t *testing.T) {
		if prID == "" {
			t.Skip("PR not created, skipping reassign test")
		}

		payload := map[string]interface{}{
			"pull_request_id": prID,
			"old_reviewer_id": "e2e_user2",
		}

		resp, err := client.post("/pullRequest/reassign", payload)
		if err != nil {
			t.Fatalf("Failed to reassign reviewer: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
			body, _ := io.ReadAll(resp.Body)
			t.Logf("Reassign status: %d, body: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("MergePullRequest", func(t *testing.T) {
		if prID == "" {
			t.Skip("PR not created, skipping merge test")
		}

		payload := map[string]interface{}{
			"pull_request_id": prID,
		}

		resp, err := client.post("/pullRequest/merge", payload)
		if err != nil {
			t.Fatalf("Failed to merge PR: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var result struct {
			PR struct {
				Status string `json:"status"`
			} `json:"pr"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result.PR.Status != "MERGED" {
			t.Errorf("Expected status MERGED, got %s", result.PR.Status)
		}
	})

	t.Run("MergePullRequest_Idempotent", func(t *testing.T) {
		if prID == "" {
			t.Skip("PR not created, skipping idempotent test")
		}

		payload := map[string]interface{}{
			"pull_request_id": prID,
		}

		resp, err := client.post("/pullRequest/merge", payload)
		if err != nil {
			t.Fatalf("Failed to merge PR second time: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected idempotent merge to return 200, got %d", resp.StatusCode)
		}
	})

	t.Run("GetStatistics", func(t *testing.T) {
		resp, err := client.get("/statistics")
		if err != nil {
			t.Fatalf("Failed to get statistics: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		var stats map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
			t.Fatalf("Failed to decode statistics: %v", err)
		}

		requiredFields := []string{"total_teams", "total_users", "total_prs"}
		for _, field := range requiredFields {
			if _, ok := stats[field]; !ok {
				t.Errorf("Statistics missing field: %s", field)
			}
		}
	})
}

func TestE2E_ErrorCases(t *testing.T) {
	client := NewTestClient()

	t.Run("CreateTeam_Duplicate", func(t *testing.T) {
		teamName := "duplicate_team_" + fmt.Sprint(time.Now().Unix())
		team := map[string]interface{}{
			"team_name": teamName,
			"members": []map[string]interface{}{
				{"user_id": "dup_user1", "username": "User1", "is_active": true},
			},
		}

		resp1, err := client.post("/team/add", team)
		if err != nil {
			t.Fatalf("Failed to create team: %v", err)
		}
		resp1.Body.Close()

		resp2, err := client.post("/team/add", team)
		if err != nil {
			t.Fatalf("Failed to send duplicate request: %v", err)
		}
		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusBadRequest && resp2.StatusCode != http.StatusConflict {
			t.Errorf("Expected 400 or 409 for duplicate team, got %d", resp2.StatusCode)
		}
	})

	t.Run("GetTeam_NotFound", func(t *testing.T) {
		resp, err := client.get("/team/get?team_name=nonexistent_team_12345")
		if err != nil {
			t.Fatalf("Failed to get team: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected 404 for nonexistent team, got %d", resp.StatusCode)
		}
	})

	t.Run("MergePR_NotFound", func(t *testing.T) {
		payload := map[string]interface{}{
			"pull_request_id": "nonexistent_pr_12345",
		}

		resp, err := client.post("/pullRequest/merge", payload)
		if err != nil {
			t.Fatalf("Failed to merge PR: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected 404 for nonexistent PR, got %d", resp.StatusCode)
		}
	})

	t.Run("CreatePR_DuplicateID", func(t *testing.T) {
		prID := "dup_pr_" + fmt.Sprint(time.Now().Unix())
		pr := map[string]interface{}{
			"pull_request_id":   prID,
			"pull_request_name": "Duplicate PR",
			"author_id":         "e2e_user1",
		}

		resp1, err := client.post("/pullRequest/create", pr)
		if err != nil {
			t.Fatalf("Failed to create PR: %v", err)
		}
		resp1.Body.Close()

		resp2, err := client.post("/pullRequest/create", pr)
		if err != nil {
			t.Fatalf("Failed to send duplicate PR request: %v", err)
		}
		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusConflict {
			t.Errorf("Expected 409 for duplicate PR, got %d", resp2.StatusCode)
		}
	})
}
