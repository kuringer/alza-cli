# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.5.0] - 2026-03-12

### Added
- `alza orders --with-items` to show item lines under each order
- `alza orders --query <text>` to search archive order history by item name
- Order-item data in JSON output when explicitly requested or when using query mode

### Changed
- `alza orders --format=json` now aligns with `--with-items` semantics for non-query mode
- Order archive fetching now supports pagination offsets for full-history search

## [0.4.0] - 2026-01-19

### Added
- `alza reviews <id>` command for product reviews and review stats
- Rating summary in `alza product` output
- Review pagination with `--offset` and `--limit`
- Stats-only review mode via `--stats`
- JSON output support for reviews

## [0.3.0] - 2026-01-13

### Added
- Clear error message when passing multiple products to quickbuy (only 1 supported)

## [0.2.2] - 2026-01-04

### Fixed
- `--no-coupon` now overrides coupons from `quickbuy.env` / env vars

## [0.2.1] - 2025-01-02

### Added
- `alza version` command to show current version
- Automatic update notification when newer version is available on GitHub

### Fixed
- CI now correctly uses Go 1.24

## [0.2.0] - 2025-01-02

### Added
- Initial open source release
- Product search with filters (`alza search`)
- Product details (`alza product`)
- Cart management (`alza cart add/remove/show/clear`)
- Favorites/lists management (`alza favorites`, `alza lists`)
- QuickBuy one-click purchasing (`alza quickbuy`)
- Token refresh from Chrome cookies (`alza token refresh`)
- Token pull from remote server (`alza token pull`)
- Remote login via SSH/VNC (`scripts/remote-login.sh`)
- Order history viewing (`alza orders`)
- User info (`alza whoami`)
- Auto-refresh expired tokens
- Comprehensive test suite (42% coverage)

### Security
- Token storage in user config directory (`~/.config/alza/`)
- Support for `--dry-run` in quickbuy
- Coupon requirement for quickbuy (prevents accidental orders)

## [0.1.0] - 2024-12-XX

### Added
- Initial development version
- Basic product search
- Cart operations
- Token management
