# Alza CLI

[![CI](https://github.com/kuringer/alza-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/kuringer/alza-cli/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

CLI tool for interacting with Alza.sk - search products, manage cart, favorites, and quick ordering.

## ⚠️ Disclaimer

**This is a hobby project.** I built it for my personal AI assistant to search and buy products on Alza programmatically.

**Important:**

- **Slovakia only** - Currently works with alza.sk. Other countries (CZ, HU, etc.) are not tested but extending should be straightforward
- **No official API** - This is reverse-engineered from browser requests. Alza doesn't provide a public API
- **Can break anytime** - If Alza changes their website, this will stop working
- **Use at your own risk** - Unofficial and unsupported

Contributions welcome, but please understand this is a side project.

## Installation

```bash
git clone https://github.com/kuringer/alza-cli.git
cd alza-cli
make build
```

### Prerequisites

1. **Chrome/Chromium** - Required for authentication (Cloudflare protection)
2. **Node.js + npm** - For reading Chrome cookies

## Authentication

The CLI uses Bearer tokens extracted from your Chrome browser session.

### Initial Setup

1. Log in to [alza.sk](https://www.alza.sk) in Chrome
2. Run token refresh:
   ```bash
   alza token refresh
   ```
   For a different Chrome profile:
   ```bash
   alza token refresh --chrome-profile "Profile 1"
   ```

### Token Auto-Refresh

Tokens expire after ~90 minutes. The CLI automatically refreshes when needed:

```
$ alza whoami
🔄 Token expired, refreshing...
✓ Token refreshed
User: John Doe (ID: 123456)
```

### Headless Servers

For servers without GUI, use the VNC-based remote login:
```bash
./scripts/remote-login.sh
```
Then connect via VNC tunnel and log in manually.

## Usage

```bash
# User info
alza whoami

# Search products
alza search "protein" -n 10

# Product details
alza product 7816725

# Cart
alza cart show
alza cart add 7816725 -q 2
alza cart remove 7816725
alza cart clear

# Favorites
alza favorites show
alza favorites add 7816725

# Lists
alza lists
alza lists items 49098230

# Order history
alza orders

# JSON output
alza cart show --format=json
```

## QuickBuy (⚠️ Dangerous)

One-click ordering to AlzaBox with saved payment card.

```bash
# Quote only (no order)
alza quickbuy 7816725 --quote --coupon SALE10

# Actual order (requires -y confirmation)
alza quickbuy 7816725 -y --coupon SALE10

# Without coupon (explicit)
alza quickbuy 7816725 -y --no-coupon
```

**Requires configuration** - create `~/.config/alza/quickbuy.env` from the example:
```bash
cp config/quickbuy.env.example ~/.config/alza/quickbuy.env
# Edit with your AlzaBox ID, payment method, etc.
```

## Configuration

Files in `~/.config/alza/`:
- `auth_token.txt` - Bearer token
- `quickbuy.env` - QuickBuy settings (optional)

Environment variables:
- `ALZA_FAVORITES_LIST` - Custom list name for favorites (default: `AGENT`)

## Development

```bash
make build    # Build binary
make test     # Run tests
make fmt      # Format code
make lint     # Run linters
```

## How It Works

1. **TLS Client** with Chrome fingerprint - bypasses Cloudflare
2. **Bearer Token** from Chrome cookies - for API authentication
3. **Reverse-engineered API** - based on browser network traffic

## License

MIT
