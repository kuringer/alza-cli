package main

import (
	"testing"

	"github.com/kuringer/alza-cli/client"
)

func TestBuildQuickbuyConfigNoCouponClearsCoupons(t *testing.T) {
	envCfg := client.QuickBuyConfig{
		AlzaBoxID:  123,
		PromoCodes: []string{"ENV1"},
	}
	cmd := QuickbuyCmd{
		Coupons:  []string{"CMD1"},
		NoCoupon: true,
	}

	cfg := buildQuickbuyConfig(&cmd, envCfg)
	if len(cfg.PromoCodes) != 0 {
		t.Fatalf("expected no coupons when --no-coupon is set, got %v", cfg.PromoCodes)
	}
	if cfg.AlzaBoxID != 123 {
		t.Fatalf("expected defaults to apply, got AlzaBoxID=%d", cfg.AlzaBoxID)
	}
}

func TestBuildQuickbuyConfigUsesEnvCouponsWhenAllowed(t *testing.T) {
	envCfg := client.QuickBuyConfig{
		PromoCodes: []string{"ENV1"},
	}

	cfg := buildQuickbuyConfig(&QuickbuyCmd{}, envCfg)
	if len(cfg.PromoCodes) != 1 || cfg.PromoCodes[0] != "ENV1" {
		t.Fatalf("expected env coupons to apply, got %v", cfg.PromoCodes)
	}
}
