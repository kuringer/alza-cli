package client

import "errors"

var (
	ErrAuthRequired = errors.New("auth required")
	ErrTokenExpired = errors.New("auth token expired or invalid")
)
