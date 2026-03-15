package core

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"
)

func TestFileCredentials_Resolve(t *testing.T) {
	// Write a temp JSON file.
	f, err := os.CreateTemp("", "skipper-creds-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	payload := map[string]string{"type": "service_account", "project_id": "test"}
	if err := json.NewEncoder(f).Encode(payload); err != nil {
		t.Fatal(err)
	}
	f.Close()

	creds := FileCredentials{Path: f.Name()}
	data, err := creds.Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty credentials data")
	}
}

func TestFileCredentials_Resolve_MissingFile(t *testing.T) {
	creds := FileCredentials{Path: "/nonexistent/path/service-account.json"}
	_, err := creds.Resolve()
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestBase64Credentials_Resolve(t *testing.T) {
	payload := `{"type":"service_account","project_id":"test"}`
	encoded := base64.StdEncoding.EncodeToString([]byte(payload))

	creds := Base64Credentials{Encoded: encoded}
	data, err := creds.Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if string(data) != payload {
		t.Errorf("got %q, want %q", string(data), payload)
	}
}

func TestBase64Credentials_Resolve_InvalidBase64(t *testing.T) {
	creds := Base64Credentials{Encoded: "!!!not-valid-base64!!!"}
	_, err := creds.Resolve()
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestServiceAccountCredentials_Resolve(t *testing.T) {
	creds := ServiceAccountCredentials{
		Type:        "service_account",
		ProjectID:   "my-project",
		PrivateKey:  "-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----",
		ClientEmail: "bot@my-project.iam.gserviceaccount.com",
	}
	data, err := creds.Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	if parsed["type"] != "service_account" {
		t.Errorf("type = %v, want service_account", parsed["type"])
	}
	if parsed["project_id"] != "my-project" {
		t.Errorf("project_id = %v, want my-project", parsed["project_id"])
	}
}
