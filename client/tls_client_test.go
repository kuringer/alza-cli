package client

import (
	"testing"
)

func TestBaseURLValue(t *testing.T) {
	if BaseURL != "https://www.alza.sk" {
		t.Errorf("BaseURL = %q, want https://www.alza.sk", BaseURL)
	}
}

func TestTLSClientSetUserID(t *testing.T) {
	c := &TLSClient{}

	c.SetUserID("12345")
	if c.GetUserID() != "12345" {
		t.Errorf("GetUserID() = %q, want 12345", c.GetUserID())
	}

	c.SetUserID("67890")
	if c.GetUserID() != "67890" {
		t.Errorf("GetUserID() = %q, want 67890", c.GetUserID())
	}
}

func TestTLSClientSetBasketID(t *testing.T) {
	c := &TLSClient{}

	c.SetBasketID("basket123")
	if c.GetBasketID() != "basket123" {
		t.Errorf("GetBasketID() = %q, want basket123", c.GetBasketID())
	}

	c.SetBasketID("basket456")
	if c.GetBasketID() != "basket456" {
		t.Errorf("GetBasketID() = %q, want basket456", c.GetBasketID())
	}
}

func TestTLSClientInitialState(t *testing.T) {
	c := &TLSClient{}

	if c.GetUserID() != "" {
		t.Errorf("GetUserID() initial = %q, want empty", c.GetUserID())
	}

	if c.GetBasketID() != "" {
		t.Errorf("GetBasketID() initial = %q, want empty", c.GetBasketID())
	}
}

func TestTLSClientSetEmptyUserID(t *testing.T) {
	c := &TLSClient{}
	c.SetUserID("12345")
	c.SetUserID("")

	if c.GetUserID() != "" {
		t.Errorf("GetUserID() after setting empty = %q, want empty", c.GetUserID())
	}
}

func TestTLSClientSetEmptyBasketID(t *testing.T) {
	c := &TLSClient{}
	c.SetBasketID("basket123")
	c.SetBasketID("")

	if c.GetBasketID() != "" {
		t.Errorf("GetBasketID() after setting empty = %q, want empty", c.GetBasketID())
	}
}

func TestTLSClientFieldsAreIsolated(t *testing.T) {
	c := &TLSClient{}

	c.SetUserID("user123")
	c.SetBasketID("basket456")

	if c.GetUserID() != "user123" {
		t.Errorf("GetUserID() = %q, want user123", c.GetUserID())
	}
	if c.GetBasketID() != "basket456" {
		t.Errorf("GetBasketID() = %q, want basket456", c.GetBasketID())
	}

	// Changing one shouldn't affect the other
	c.SetUserID("user789")
	if c.GetBasketID() != "basket456" {
		t.Errorf("GetBasketID() changed unexpectedly to %q", c.GetBasketID())
	}
}

func TestTLSClientDebugField(t *testing.T) {
	c := &TLSClient{debug: true}
	if !c.debug {
		t.Error("debug field = false, want true")
	}

	c2 := &TLSClient{debug: false}
	if c2.debug {
		t.Error("debug field = true, want false")
	}
}

func TestTLSClientAuthTokenField(t *testing.T) {
	c := &TLSClient{authToken: "Bearer test123"}
	if c.authToken != "Bearer test123" {
		t.Errorf("authToken = %q, want Bearer test123", c.authToken)
	}
}
