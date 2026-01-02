# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
