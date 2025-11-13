package analyzer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	config := ClientConfig{
		BaseURL: "http://localhost:8000",
		Timeout: 10 * time.Second,
	}

	client := NewClient(config)

	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}

	if client.baseURL != config.BaseURL {
		t.Errorf("Expected baseURL %s, got %s", config.BaseURL, client.baseURL)
	}

	if client.httpClient.Timeout != config.Timeout {
		t.Errorf("Expected timeout %v, got %v", config.Timeout, client.httpClient.Timeout)
	}
}

func TestNewClient_DefaultTimeout(t *testing.T) {
	config := ClientConfig{
		BaseURL: "http://localhost:8000",
	}

	client := NewClient(config)

	expectedTimeout := 30 * time.Second
	if client.httpClient.Timeout != expectedTimeout {
		t.Errorf("Expected default timeout %v, got %v", expectedTimeout, client.httpClient.Timeout)
	}
}

func TestAnalyzeContent_Success_Flagged(t *testing.T) {
	// Create a test server that returns a flagged result
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/api/v1/analyze/content" {
			t.Errorf("Expected path /api/v1/analyze/content, got %s", r.URL.Path)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Return a flagged result
		result := AnalysisResult{
			IsFlagged:  true,
			FlagReason: "Toxic content detected",
			Scores: map[string]float64{
				"toxicity": 0.95,
				"sexual":   0.0,
				"violence": 0.0,
				"spam":     0.0,
			},
			Confidence: 0.98,
			Timestamp:  "2025-10-09T12:00:00Z",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	ctx := context.Background()
	result, err := client.AnalyzeContent(ctx, "This is toxic content")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !result.IsFlagged {
		t.Error("Expected content to be flagged")
	}

	if result.FlagReason != "Toxic content detected" {
		t.Errorf("Expected flag reason 'Toxic content detected', got '%s'", result.FlagReason)
	}

	if result.Scores["toxicity"] != 0.95 {
		t.Errorf("Expected toxicity score 0.95, got %f", result.Scores["toxicity"])
	}
}

func TestAnalyzeContent_Success_Approved(t *testing.T) {
	// Create a test server that returns an approved result
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := AnalysisResult{
			IsFlagged: false,
			Scores: map[string]float64{
				"toxicity": 0.0,
				"sexual":   0.0,
				"violence": 0.0,
				"spam":     0.0,
			},
			Confidence: 0.95,
			Timestamp:  "2025-10-09T12:00:00Z",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	ctx := context.Background()
	result, err := client.AnalyzeContent(ctx, "This is benign content")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.IsFlagged {
		t.Error("Expected content to not be flagged")
	}

	if result.Scores["toxicity"] != 0.0 {
		t.Errorf("Expected toxicity score 0.0, got %f", result.Scores["toxicity"])
	}
}

func TestAnalyzeContent_ServerError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	ctx := context.Background()
	_, err := client.AnalyzeContent(ctx, "Test content")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	expectedError := "analyzer service returned status 500"
	if !contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedError, err.Error())
	}
}

func TestAnalyzeContent_Timeout(t *testing.T) {
	// Create a test server that delays the response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		Timeout: 100 * time.Millisecond, // Very short timeout
	})

	ctx := context.Background()
	_, err := client.AnalyzeContent(ctx, "Test content")

	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}
}

func TestAnalyzeContent_InvalidJSON(t *testing.T) {
	// Create a test server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	ctx := context.Background()
	_, err := client.AnalyzeContent(ctx, "Test content")

	if err == nil {
		t.Fatal("Expected JSON parsing error, got nil")
	}

	expectedError := "failed to unmarshal response"
	if !contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedError, err.Error())
	}
}

func TestHealthCheck_Success(t *testing.T) {
	// Create a test server that returns healthy status
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("Expected path /health, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	ctx := context.Background()
	err := client.HealthCheck(ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestHealthCheck_Failure(t *testing.T) {
	// Create a test server that returns unhealthy status
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"unhealthy"}`))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	ctx := context.Background()
	err := client.HealthCheck(ctx)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	expectedError := "health check failed with status: 503"
	if !contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedError, err.Error())
	}
}

func TestAnalyzeContent_ContextCancellation(t *testing.T) {
	// Create a test server that delays the response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.AnalyzeContent(ctx, "Test content")

	if err == nil {
		t.Fatal("Expected context cancellation error, got nil")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}


