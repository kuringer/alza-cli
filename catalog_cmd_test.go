package main

import "testing"

// Regression (papercut d47bb711): `alza search` failed with "auth required"
// when the token was missing or expired. Search must not need a token.
func TestSearchClientWorksWithoutToken(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // no ~/.config/alza/auth_token.txt

	cl, err := newSearchClient(&Globals{})
	if err != nil {
		t.Fatalf("newSearchClient without token: %v", err)
	}
	if cl == nil {
		t.Fatal("newSearchClient returned nil client")
	}
}
