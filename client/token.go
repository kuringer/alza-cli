package client

import (
	"os"
	"path/filepath"
)

// SaveToken saves the token to config file.
func SaveToken(token string) error {
	path, err := TokenPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(token), 0600)
}
