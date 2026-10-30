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
alza orders --with-items
alza orders --query "fólia"

# JSON output
alza cart show --format=json
alza orders --with-items --format=json
alza orders --query "fólia" --format=json
```

Notes:
- `alza orders --query` searches archive history items only
- `--query` always shows matching item lines - `--with-items` is only for non-query mode
- non-query JSON includes `items` only with `--with-items`
- query JSON includes `orders`, `totalCount`, `historyCount`, `query`, `searchesArchiveOnly`

## Deals: is the discount real?

Alza often shows a big discount against a crossed-out price. `alza deals`
compares the Alza price with the market (Heureka.sk) and with your own price
history.

```bash
alza deals check 1234567              # Alza vs Heureka vs history, one verdict
alza deals check 1234567 --no-heureka # history only, no browser
alza deals check 1234567 --format=json
alza deals snapshot --id 1234567      # store today's prices (favorites + --id)
alza deals scan --favorites           # check all favorites, biggest saving first
alza deals browse                     # Black Friday: top 20 deals checked on Heureka
alza deals browse --check 0 --query lego   # pre-filter only, no browser
alza deals browse --url https://www.alza.sk/notebooky/18842920.htm --sort sales --min-discount 10
```

`deals browse` reads a whole deal listing, not only your favorites. It loads
`--pages` listing pages (default 10 x 24 products, sorted by discount), keeps
products that pass the cheap filter (`--min-discount 20`, `--min-reviews 10`,
`--min-rating`, `--min-price`, `--max-price`, `--query`; used "bazar" items
only with `--used`), ranks them by the saving Alza claims in EUR and checks the
top `--check` (default 20) on Heureka. The table is sorted by real saving vs
market and shows the claimed % next to the real % (negative = Alza is more
expensive). A `?` marks an uncertain Heureka match; `[bazár]` marks used items
(`--used`). If every Heureka lookup fails, the command exits with code 1. Listing pages need no
login; with a token the check also uses part numbers (more certain matches).
20 products take about 1 minute (one headless browser, 3 pages at once).

Verdicts: `skutočná zľava` (Alza <= cheapest market offer incl. shipping, or
<= 95 % of its own 90-day median), `bežná cena` (within 5 % of the market
minimum), `drahšie ako trh` (more than 5 % above it), `neisté porovnanie`
(uncertain Heureka match or market price without a known shop; the market
result is shown only as a hint). The output lists missing data (no Heureka
match, uncertain match, less than 7 days of history).

Notes:
- Heureka blocks plain HTTP clients, so pages are read in headless Chromium
  (Playwright, no window). First use installs it into `~/.cache/alza/headless`
  (~100 MB, needs Node.js). `ALZA_BROWSER_RUNNER=surf` uses the surf CLI
  instead (`ALZA_SURF_BIN` = binary path).
- Shipping prices from Heureka are approximate (free-shipping thresholds).
- History: `~/.config/alza/price-history.jsonl` (override `ALZA_PRICE_HISTORY`),
  one record per product per day. `check` and `scan` also record today's price
  (`--no-record` to skip). Run `snapshot` daily (cron) to build history.
- Matching uses the Alza part number first, then the name. A weak match is
  marked "uncertain match".
- `browse` only reads your price history (it does not record listing prices).

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
