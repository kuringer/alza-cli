package client

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ConfigDir returns ~/.config/alza.
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "alza"), nil
}

// TokenPath returns the path to auth_token.txt.
func TokenPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "auth_token.txt"), nil
}

// QuickbuyEnvPath returns the path to quickbuy.env.
func QuickbuyEnvPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "quickbuy.env"), nil
}

// QuickbuyConfigFromEnvFile loads quickbuy.env. Missing file returns empty config.
func QuickbuyConfigFromEnvFile(path string) (QuickBuyConfig, error) {
	if path == "" {
		var err error
		path, err = QuickbuyEnvPath()
		if err != nil {
			return QuickBuyConfig{}, err
		}
	}

	data, err := readEnvFile(path)
	if err != nil {
		return QuickBuyConfig{}, err
	}
	if len(data) == 0 {
		return QuickBuyConfig{}, nil
	}

	cfg := QuickBuyConfig{}
	if val, ok := data["ALZA_QUICKBUY_ALZABOX_ID"]; ok {
		parsed, err := strconv.Atoi(val)
		if err != nil {
			return QuickBuyConfig{}, errorForEnv(path, "ALZA_QUICKBUY_ALZABOX_ID")
		}
		cfg.AlzaBoxID = parsed
	}
	if val, ok := data["ALZA_QUICKBUY_DELIVERY_ID"]; ok {
		parsed, err := strconv.Atoi(val)
		if err != nil {
			return QuickBuyConfig{}, errorForEnv(path, "ALZA_QUICKBUY_DELIVERY_ID")
		}
		cfg.DeliveryID = parsed
	}
	if val, ok := data["ALZA_QUICKBUY_PAYMENT_ID"]; ok {
		cfg.PaymentID = val
	}
	if val, ok := data["ALZA_QUICKBUY_CARD_ID"]; ok {
		cfg.CardID = val
	}
	if val, ok := data["ALZA_QUICKBUY_VISITOR_ID"]; ok {
		cfg.VisitorID = val
	}
	if val, ok := data["ALZA_QUICKBUY_ALZAPLUS"]; ok && parseEnvBool(val) {
		cfg.IsAlzaPlus = true
	}
	if val, ok := data["ALZA_QUICKBUY_COUPON"]; ok {
		cfg.PromoCodes = normalizePromoCodes([]string{val})
	}

	return cfg, nil
}

func errorForEnv(path, key string) error {
	return fmt.Errorf("invalid %s in %s", key, path)
}

func readEnvFile(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	out := map[string]string{}
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key != "" {
			out[key] = value
		}
	}
	return out, nil
}

func parseEnvBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
