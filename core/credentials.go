package core

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
)

// Credentials resolves to a Google service account JSON map.
type Credentials interface {
	Resolve() ([]byte, error)
}

// FileCredentials reads a service account JSON from a file path.
type FileCredentials struct {
	Path string
}

func (c FileCredentials) Resolve() ([]byte, error) {
	data, err := os.ReadFile(c.Path)
	if err != nil {
		return nil, fmt.Errorf("skipper: cannot read credentials file %q: %w", c.Path, err)
	}
	return data, nil
}

// Base64Credentials decodes a base64-encoded service account JSON string.
// Useful for passing credentials via environment variables in CI/CD.
type Base64Credentials struct {
	Encoded string
}

func (c Base64Credentials) Resolve() ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(c.Encoded)
	if err != nil {
		return nil, fmt.Errorf("skipper: cannot decode base64 credentials: %w", err)
	}
	return data, nil
}

// ServiceAccountCredentials holds the parsed service account fields inline.
type ServiceAccountCredentials struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
}

func (c ServiceAccountCredentials) Resolve() ([]byte, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("skipper: cannot marshal service account credentials: %w", err)
	}
	return data, nil
}
