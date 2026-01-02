package client

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigDir(t *testing.T) {
	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error: %v", err)
	}

	if dir == "" {
		t.Error("ConfigDir() returned empty string")
	}

	if !filepath.IsAbs(dir) {
		t.Errorf("ConfigDir() = %q, expected absolute path", dir)
	}

	if !strings.Contains(dir, ".config") || !strings.Contains(dir, "alza") {
		t.Errorf("ConfigDir() = %q, expected to contain .config/alza", dir)
	}
}

func TestTokenPath(t *testing.T) {
	path, err := TokenPath()
	if err != nil {
		t.Fatalf("TokenPath() error: %v", err)
	}

	if filepath.Base(path) != "auth_token.txt" {
		t.Errorf("TokenPath() = %q, expected auth_token.txt", path)
	}
}

func TestQuickbuyEnvPath(t *testing.T) {
	path, err := QuickbuyEnvPath()
	if err != nil {
		t.Fatalf("QuickbuyEnvPath() error: %v", err)
	}

	if filepath.Base(path) != "quickbuy.env" {
		t.Errorf("QuickbuyEnvPath() = %q, expected quickbuy.env", path)
	}
}

func TestReadEnvFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantKeys []string
		wantVals []string
	}{
		{
			name:     "simple key-value",
			content:  "KEY=value",
			wantKeys: []string{"KEY"},
			wantVals: []string{"value"},
		},
		{
			name:     "multiple key-values",
			content:  "KEY1=value1\nKEY2=value2",
			wantKeys: []string{"KEY1", "KEY2"},
			wantVals: []string{"value1", "value2"},
		},
		{
			name:     "with comments",
			content:  "# comment\nKEY=value\n# another comment",
			wantKeys: []string{"KEY"},
			wantVals: []string{"value"},
		},
		{
			name:     "empty lines",
			content:  "KEY1=value1\n\nKEY2=value2\n",
			wantKeys: []string{"KEY1", "KEY2"},
			wantVals: []string{"value1", "value2"},
		},
		{
			name:     "value with equals sign",
			content:  "KEY=val=ue",
			wantKeys: []string{"KEY"},
			wantVals: []string{"val=ue"},
		},
		{
			name:     "whitespace trimmed",
			content:  "  KEY  =  value  ",
			wantKeys: []string{"KEY"},
			wantVals: []string{"value"},
		},
		{
			name:     "empty file",
			content:  "",
			wantKeys: []string{},
			wantVals: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "env_test_*.env")
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tt.content); err != nil {
				t.Fatalf("failed to write to temp file: %v", err)
			}
			tmpFile.Close()

			got, err := readEnvFile(tmpFile.Name())
			if err != nil {
				t.Fatalf("readEnvFile() error: %v", err)
			}

			if len(got) != len(tt.wantKeys) {
				t.Errorf("readEnvFile() returned %d keys, want %d", len(got), len(tt.wantKeys))
			}

			for i, key := range tt.wantKeys {
				if val, ok := got[key]; !ok {
					t.Errorf("readEnvFile() missing key %q", key)
				} else if val != tt.wantVals[i] {
					t.Errorf("readEnvFile()[%q] = %q, want %q", key, val, tt.wantVals[i])
				}
			}
		})
	}
}

func TestReadEnvFileMissing(t *testing.T) {
	got, err := readEnvFile("/nonexistent/path/file.env")
	if err != nil {
		t.Errorf("readEnvFile() error for missing file: %v", err)
	}
	if got != nil {
		t.Errorf("readEnvFile() = %v for missing file, want nil", got)
	}
}

func TestParseEnvBool(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"1", true},
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"yes", true},
		{"YES", true},
		{"y", true},
		{"Y", true},
		{"on", true},
		{"ON", true},
		{"0", false},
		{"false", false},
		{"no", false},
		{"n", false},
		{"off", false},
		{"", false},
		{"anything", false},
		{"  true  ", true},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got := parseEnvBool(tt.value)
			if got != tt.want {
				t.Errorf("parseEnvBool(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestQuickbuyConfigFromEnvFile(t *testing.T) {
	content := `ALZA_QUICKBUY_ALZABOX_ID=12345
ALZA_QUICKBUY_DELIVERY_ID=67890
ALZA_QUICKBUY_PAYMENT_ID=216
ALZA_QUICKBUY_CARD_ID=card123
ALZA_QUICKBUY_VISITOR_ID=visitor456
ALZA_QUICKBUY_ALZAPLUS=true
ALZA_QUICKBUY_COUPON=PROMO1, PROMO2`

	tmpFile, err := os.CreateTemp("", "quickbuy_test_*.env")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	cfg, err := QuickbuyConfigFromEnvFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("QuickbuyConfigFromEnvFile() error: %v", err)
	}

	if cfg.AlzaBoxID != 12345 {
		t.Errorf("AlzaBoxID = %d, want 12345", cfg.AlzaBoxID)
	}
	if cfg.DeliveryID != 67890 {
		t.Errorf("DeliveryID = %d, want 67890", cfg.DeliveryID)
	}
	if cfg.PaymentID != "216" {
		t.Errorf("PaymentID = %q, want 216", cfg.PaymentID)
	}
	if cfg.CardID != "card123" {
		t.Errorf("CardID = %q, want card123", cfg.CardID)
	}
	if cfg.VisitorID != "visitor456" {
		t.Errorf("VisitorID = %q, want visitor456", cfg.VisitorID)
	}
	if !cfg.IsAlzaPlus {
		t.Error("IsAlzaPlus = false, want true")
	}
	if len(cfg.PromoCodes) != 2 || cfg.PromoCodes[0] != "PROMO1" || cfg.PromoCodes[1] != "PROMO2" {
		t.Errorf("PromoCodes = %v, want [PROMO1 PROMO2]", cfg.PromoCodes)
	}
}

func TestQuickbuyConfigFromEnvFileInvalidInt(t *testing.T) {
	content := `ALZA_QUICKBUY_ALZABOX_ID=notanumber`

	tmpFile, err := os.CreateTemp("", "quickbuy_test_*.env")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	_, err = QuickbuyConfigFromEnvFile(tmpFile.Name())
	if err == nil {
		t.Error("QuickbuyConfigFromEnvFile() expected error for invalid int")
	}
}

func TestQuickbuyConfigFromEnvFileMissing(t *testing.T) {
	cfg, err := QuickbuyConfigFromEnvFile("/nonexistent/path/quickbuy.env")
	if err != nil {
		t.Fatalf("QuickbuyConfigFromEnvFile() error for missing file: %v", err)
	}

	// Should return empty config
	if cfg.AlzaBoxID != 0 || cfg.DeliveryID != 0 {
		t.Errorf("Expected empty config for missing file, got: %+v", cfg)
	}
}

func TestQuickbuyConfigFromEnvFileInvalidDeliveryID(t *testing.T) {
	content := `ALZA_QUICKBUY_DELIVERY_ID=notanumber`

	tmpFile, err := os.CreateTemp("", "quickbuy_test_*.env")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	_, err = QuickbuyConfigFromEnvFile(tmpFile.Name())
	if err == nil {
		t.Error("QuickbuyConfigFromEnvFile() expected error for invalid delivery ID")
	}
}

func TestQuickbuyConfigFromEnvFileEmptyPath(t *testing.T) {
	// When path is empty, it should use default path
	// This will likely fail because the default path doesn't exist in test env
	// but we're testing the path resolution logic
	tmpDir, err := os.MkdirTemp("", "alza_config_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cfg, err := QuickbuyConfigFromEnvFile("")
	if err != nil {
		t.Fatalf("QuickbuyConfigFromEnvFile(\"\") error: %v", err)
	}

	// Should return empty config since file doesn't exist
	if cfg.AlzaBoxID != 0 {
		t.Errorf("Expected empty config, got: %+v", cfg)
	}
}

func TestConfigDirWithValidHome(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "alza_config_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error: %v", err)
	}

	expected := filepath.Join(tmpDir, ".config", "alza")
	if dir != expected {
		t.Errorf("ConfigDir() = %q, want %q", dir, expected)
	}
}

func TestTokenPathWithValidHome(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "alza_config_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	path, err := TokenPath()
	if err != nil {
		t.Fatalf("TokenPath() error: %v", err)
	}

	expected := filepath.Join(tmpDir, ".config", "alza", "auth_token.txt")
	if path != expected {
		t.Errorf("TokenPath() = %q, want %q", path, expected)
	}
}

func TestQuickbuyEnvPathWithValidHome(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "alza_config_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	path, err := QuickbuyEnvPath()
	if err != nil {
		t.Fatalf("QuickbuyEnvPath() error: %v", err)
	}

	expected := filepath.Join(tmpDir, ".config", "alza", "quickbuy.env")
	if path != expected {
		t.Errorf("QuickbuyEnvPath() = %q, want %q", path, expected)
	}
}

func TestReadEnvFilePermissionError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test as root")
	}

	tmpDir, err := os.MkdirTemp("", "alza_perm_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file
	tmpFile := filepath.Join(tmpDir, "test.env")
	if err := os.WriteFile(tmpFile, []byte("KEY=value"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Remove read permission
	if err := os.Chmod(tmpFile, 0000); err != nil {
		t.Fatalf("failed to chmod: %v", err)
	}

	_, err = readEnvFile(tmpFile)
	if err == nil {
		t.Error("readEnvFile() expected error for unreadable file")
	}

	// Restore permissions for cleanup
	os.Chmod(tmpFile, 0644)
}
