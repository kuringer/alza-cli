package client

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveToken(t *testing.T) {
	// Create a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "alza_token_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override home for this test
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	testToken := "Bearer test_token_123"

	err = SaveToken(testToken)
	if err != nil {
		t.Fatalf("SaveToken() error: %v", err)
	}

	// Verify the token was saved
	expectedPath := filepath.Join(tmpDir, ".config", "alza", "auth_token.txt")
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("failed to read saved token: %v", err)
	}

	if string(data) != testToken {
		t.Errorf("saved token = %q, want %q", string(data), testToken)
	}

	// Verify permissions
	info, err := os.Stat(expectedPath)
	if err != nil {
		t.Fatalf("failed to stat token file: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("token file permissions = %o, want 0600", perm)
	}
}

func TestSaveTokenCreatesDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "alza_token_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Config dir doesn't exist yet
	configDir := filepath.Join(tmpDir, ".config", "alza")
	if _, err := os.Stat(configDir); !os.IsNotExist(err) {
		t.Fatalf("config dir should not exist initially")
	}

	err = SaveToken("test_token")
	if err != nil {
		t.Fatalf("SaveToken() error: %v", err)
	}

	// Now it should exist
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Error("SaveToken() should create config directory")
	}
}

func TestSaveTokenOverwrites(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "alza_token_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Save first token
	err = SaveToken("token_v1")
	if err != nil {
		t.Fatalf("SaveToken() error: %v", err)
	}

	// Save second token
	err = SaveToken("token_v2")
	if err != nil {
		t.Fatalf("SaveToken() error: %v", err)
	}

	// Read and verify
	path, _ := TokenPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read token: %v", err)
	}

	if string(data) != "token_v2" {
		t.Errorf("saved token = %q, want token_v2", string(data))
	}
}
