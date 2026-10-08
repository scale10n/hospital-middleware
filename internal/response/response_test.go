package response_test

import (
	"encoding/json"
	"testing"

	"hospital-middleware/internal/response"
)

type sampleData struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func TestResponse_Success_WithMessage(t *testing.T) {
	data := sampleData{ID: "123", Name: "Test"}
	res := response.Success(data, "operation completed")

	if res.Status != response.StatusSuccess {
		t.Errorf("expected status 'success', got %q", res.Status)
	}
	if res.Message != "operation completed" {
		t.Errorf("expected message 'operation completed', got %q", res.Message)
	}
	if res.Data.ID != "123" || res.Data.Name != "Test" {
		t.Errorf("unexpected data: %+v", res.Data)
	}

	bytes, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var parsed response.Response[sampleData]
	if err := json.Unmarshal(bytes, &parsed); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if parsed.Status != "success" || parsed.Message != "operation completed" || parsed.Data.ID != "123" {
		t.Errorf("parsed response does not match: %+v", parsed)
	}
}

func TestResponse_Success_WithoutMessage(t *testing.T) {
	data := map[string]int{"count": 42}
	res := response.Success(data)

	if res.Status != response.StatusSuccess {
		t.Errorf("expected status 'success', got %q", res.Status)
	}
	if res.Message != "" {
		t.Errorf("expected empty message, got %q", res.Message)
	}

	bytes, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(bytes, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify omitempty behavior for empty message
	if _, exists := parsed["message"]; exists {
		t.Errorf("expected 'message' key to be omitted when empty")
	}
}
