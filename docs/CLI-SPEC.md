# CLI Špecifikácia: `alza`

## 1. Základné info

| | |
|---|---|
| **Názov** | `alza` |
| **Účel** | CLI pre interakciu s Alza.sk - vyhľadávanie, košík, obľúbené, zoznamy |
| **Jazyk** | Go + Kong framework |
| **Auth** | Bearer token z browser session |

## 2. USAGE

```
alza [global-flags] <command> [subcommand] [args] [flags]
```

## 3. Implementované príkazy

### Info
| Command | Popis | Status |
|---------|-------|--------|
| `alza whoami` | Info o prihlásenom userovi | ✅ |

### Token
| Command | Popis | Status |
|---------|-------|--------|
| `alza token refresh` | Refresh Bearer token z Chrome cookies | ✅ |
| `alza token pull --from <ssh>` | Stiahne Bearer token zo servera | ✅ |

### Vyhľadávanie
| Command | Popis | Status |
|---------|-------|--------|
| `alza search <query>` | Vyhľadá produkty | ✅ |
| `alza search <query> -n 5` | Limit výsledkov | ✅ |

### Produkt
| Command | Popis | Status |
|---------|-------|--------|
| `alza product <id>` | Detail produktu | ✅ |

### Košík
| Command | Popis | Status |
|---------|-------|--------|
| `alza cart` / `alza cart show` | Zobrazí košík | ✅ |
| `alza cart add <id>` | Pridá produkt | ✅ |
| `alza cart add <id> -q 2` | Pridá s množstvom | ✅ |
| `alza cart remove <id>` | Odstráni produkt | ✅ |
| `alza cart clear` | Vyprázdni košík | ✅ |

### Obľúbené
| Command | Popis | Status |
|---------|-------|--------|
| `alza favorites` / `alza favorites show` | Zobrazí obľúbené | ✅ |
| `alza favorites add <id>` | Pridá do obľúbených | ✅ |
| `alza favorites remove <id>` | Odstráni | ✅ |

### Zoznamy (Commodity Lists)
| Command | Popis | Status |
|---------|-------|--------|
| `alza lists` | Zobrazí všetky zoznamy | ✅ |
| `alza lists items <id>` | Položky v zozname | ✅ |
| `alza lists create <name>` | Vytvorí zoznam | ✅ |

### Objednávky
| Command | Popis | Status |
|---------|-------|--------|
| `alza orders` | Aktívne + archívne objednávky | ✅ |

## 4. Globálne flagy

| Flag | Popis | Default |
|------|-------|---------|
| `-h, --help` | Zobrazí help | |
| `--format` | Output format | `text` |
| `-d, --debug` | Debug output | false |

## 5. Output formáty

### Text (default)
```
Cart (2 items):

1. [7191542] Kreatín GymBeam 100% kreatín monohydrát 250 g
   Price: 12,90 € | Qty: 1
   https://www.alza.sk/sport/gymbeam-kreatin-d7191542.htm
```

### JSON (`--format=json`)
```json
[
  {
    "productId": 7191542,
    "count": 1,
    "basketItemId": 898234016,
    "name": "Kreatín GymBeam 100% kreatín monohydrát 250 g",
    "price": "12,90 €",
    "imageUrl": "https://image.alza.cz/...",
    "url": "https://www.alza.sk/..."
  }
]
```

## 6. Autentifikácia

### Bearer Token
Token sa ukladá v `~/.config/alza/auth_token.txt`

### Auto-refresh tokenu

Token sa **automaticky refreshne** ak expiroval (vyžaduje prihlásený Chrome):
```
$ alza cart show
🔄 Token expiroval, skúšam automatický refresh...
✓ Token refreshnutý, pokračujem...
Cart (2 items): ...
```

### Manuálny refresh
```bash
alza token refresh
```
Používa Chrome cookies (vyžaduje Node + npm). Ak máš iný profil:
```bash
alza token refresh --chrome-profile "Profile 1"
```

### Keď refresh zlyhá (login required)
- Desktop: otvor Chrome/Chromium, prihlás sa do alza.sk a skús príkaz znovu
- Headless: použi `./scripts/remote-login.sh` (VNC), prihlás sa, potom Enter → refresh

### Prečo stále `tls-client`?
Cloudflare blokuje bežných HTTP klientov. `tls-client` spoofuje Chrome TLS fingerprint pre API volania.

## 6.1 Favorites mapovanie

`alza favorites` pracuje s custom listom (type 0) s názvom `AGENT` (fallback `AGENTS`).
Iný názov môžeš nastaviť cez env `ALZA_FAVORITES_LIST`.

## 7. Architektúra

```
┌─────────────┐     ┌──────────────────┐     ┌─────────────┐
│   alza CLI  │────▶│   tls-client     │────▶│  alza.sk    │
│   (Go)      │     │ (Chrome fingerp.)│     │ (Cloudflare)│
└─────────────┘     └──────────────────┘     └─────────────┘
```

- **tls-client:** Go knižnica ktorá spoofuje Chrome TLS fingerprint
- Obchádza Cloudflare bez potreby browsera
- Bearer token z Chrome cookies (po jednorazovom prihlásení v browseri)

## 8. API Endpointy

| Akcia | Endpoint | Method |
|-------|----------|--------|
| User info | `/services/restservice.svc/v1/getCommodityLists` | GET |
| User status | `/api/users/{id}/statusSummary` | GET |
| Search | `webapi.alza.cz/api/users/{id}/search/whisperer/v1/whisper` | GET |
| Product detail | `/api/router/legacy/catalog/product/{id}?country=SK&electronicContentOnly=False` | GET |
| Add to cart | `/Services/EShopService.svc/OrderCommodity` | POST |
| Update/Remove cart item | `/Services/EShopService.svc/OrderUpdate?country=SK` | POST |
| Get cart items | `/api/v1/anonymous/baskets/{id}/checkout/cart/items` | GET |
| Cart preview | `/api/basket/{id}/preview` | GET |
| Empty cart | `/api/v1/anonymous/baskets/{id}/checkout/cart/items` | DELETE |
| Get lists | `/services/restservice.svc/v1/getCommodityLists` | GET |
| Get list items | `/services/restservice.svc/v1/getCommodityLists/{id}` | GET |
| Add to favorites | `/api/v1/users/{id}/commodityList/items` | POST |

## 9. Príklady

```bash
# Info o userovi
alza whoami

# Refresh token z Chrome cookies
alza token refresh

# Pull token zo servera
alza token pull --from <ssh-host>

# Vyhľadať kreatín (max 5 výsledkov)
alza search "kreatin" -n 5

# Pridať do košíka
alza cart add 7191542

# Zobraziť košík
alza cart show

# JSON output
alza cart show --format=json

# Odstrániť z košíka
alza cart remove 7191542

# Vyprázdniť košík
alza cart clear

# Zobraziť zoznamy
alza lists

# Položky v zozname
alza lists items 44367457

# Pridať do obľúbených
alza favorites add 7191542

# Detail produktu
alza product 7816725

# Debug mode
alza -d cart show
```

## 10. Konfigurácia

```
~/.config/alza/
├── auth_token.txt    # Bearer token
```

Cache:
```
~/.cache/alza/chromecookies/   # node_modules pre chrome-cookies-secure
```

## 11. Build

```bash
go build -o alza .
```

## 12. Quick Buy (⚠️ DANGEROUS)

Rýchle objednanie do AlzaBoxu s uloženou kartou.

### Kupón je povinný

Quickbuy vyžaduje kupón. Bez kupónu treba explicitne `--no-coupon`:

```bash
# S kupónom (štandard)
alza quickbuy 7816725 -y --coupon VYPREDAJ15

# Bez kupónu (explicitne)
alza quickbuy 7816725 -y --no-coupon

# Len cenová ponuka
alza quickbuy 7816725 --quote --coupon ZLAVA10

# S množstvom
alza quickbuy 7816725 -q 2 -y --coupon VYPREDAJ15
```

**Vyžaduje konfiguráciu (flags alebo env):**
- `ALZA_QUICKBUY_ALZABOX_ID`
- `ALZA_QUICKBUY_DELIVERY_ID`
- `ALZA_QUICKBUY_PAYMENT_ID`
- `ALZA_QUICKBUY_CARD_ID`
- `ALZA_QUICKBUY_VISITOR_ID`
- `ALZA_QUICKBUY_ALZAPLUS` (voliteľné)
- `ALZA_QUICKBUY_COUPON` (voliteľné v env, viac kódov cez čiarku)

**Poznámka:** pre `--quote` stačí `ALZA_QUICKBUY_ALZABOX_ID`, `ALZA_QUICKBUY_DELIVERY_ID` a `ALZA_QUICKBUY_PAYMENT_ID`.

Voliteľne môžeš použiť config súbor:
```
~/.config/alza/quickbuy.env
```
Vzorka v `config/quickbuy.env.example`.

## 13. Changelog

Pre históriu zmien pozri [CHANGELOG.md](../CHANGELOG.md).
