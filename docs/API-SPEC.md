# Alza API Specification

Reverse-engineered API dokumentácia pre Alza.sk

## ⚠️ Dôležité poznámky

### Cloudflare Protection
- Všetky Alza domény sú chránené Cloudflare
- **Priame curl/fetch volania nefungujú** - vracajú JS challenge
- **Riešenie:** Použiť Playwright/Puppeteer s persistent browser session
- Cookies z browser session sa dajú extrahovať a použiť

### Domény
| Doména | Účel |
|--------|------|
| `www.alza.sk` | Hlavné API pre SK |
| `webapi.alza.cz` | Zdieľané API (CZ/SK) |
| `identity.alza.cz` | OAuth/OIDC autentifikácia |
| `chatbotapi.alza.cz` | Chatbot API |
| `logapi.alza.cz` | Analytics (ignorovať) |

---

## 1. Visitor/Session

### Get Visitor Status
Získa stav košíka pre visitor ID.

```
GET /api/visitors/{visitorId}/statusSummary
Host: www.alza.sk
```

**Response:**
```json
{
  "self": {
    "href": "https://www.alza.sk/api/visitors/{visitorId}/statusSummary",
    "appLink": "visitorStatusSummary",
    "enabled": true
  },
  "basketProductsCount": 1,
  "basketPreviewAction": {
    "href": "https://www.alza.sk/api/basket/{basketId}/preview",
    "appLink": "basketPreview",
    "enabled": true
  }
}
```

**Poznámka:** `visitorId` je UUID uložený v cookies (napr. `1a4e7418-81f0-4214-8a5a-401774bc9b0b`)

---

## 2. Search / Autocomplete

### Whisper (Autocomplete)
Vyhľadávanie s autocomplete.

```
GET /api/anonymous/search/whisperer/v1/whisper
Host: webapi.alza.cz
```

**Parameters:**
| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `searchTerm` | string | ✓ | Hľadaný výraz |
| `country` | string | ✓ | `SK` alebo `CZ` |
| `eshopUrl` | string | ✓ | `https://www.alza.sk/` |
| `visitor` | string | ✓ | Visitor UUID |

**Response:**
```json
{
  "articles": [...],
  "categories": [
    {
      "clickAction": {
        "webLink": "https://www.alza.sk/sport/kreatin/18862660.htm",
        "href": "https://www.alza.sk/Services/RestService.svc/v1/category/18862660",
        "appLink": "catalogCategory",
        "name": "Kreatín"
      }
    }
  ],
  "products": [
    {
      "id": 7191542,
      "name": "GymBeam 100% kreatín monohydrát 250 g",
      "price": "12,90 €",
      "imageUrl": "...",
      "clickAction": {...}
    }
  ]
}
```

### Search (Full Results)
Plnohodnotné vyhľadávanie s cenami a dostupnosťou.

```
POST /Services/RestService.svc/v5/search
Host: www.alza.sk
Content-Type: application/json
```

**Request Body (minimum):**
```json
{
  "searchTerm": "kreatin"
}
```

**Response (skrátené):**
```json
{
  "total": 51,
  "data2": [
    {
      "id": 5275186,
      "code": "SPTami0032",
      "name": "Amix Nutrition Creatine monohydrate, powder, 500 g",
      "img": "https://image.alza.cz/...",
      "url": "https://www.alza.sk/sport/amix-nutrition-creatine-monohydrate-powder-500-g-d5275186.htm",
      "price": "18,90 €",
      "priceNoCurrency": 18.90,
      "avail": "Na sklade > 10 ks"
    }
  ]
}
```

### Empty Search (Populárne)
Získa populárne vyhľadávania.

```
GET /api/anonymous/search/whisperer/v1/emptySearch
Host: webapi.alza.cz
```

---

## 3. Add to Cart

### Order Commodity (Add to Cart)
Pridá produkt do košíka.

```
POST /Services/EShopService.svc/OrderCommodity
Host: www.alza.sk
Content-Type: application/json
```

**Request Body:**
```json
{
  "id": 7191542,
  "count": 1,
  "warranty": null,
  "insurance": null,
  "replacement": null,
  "hooks": null,
  "src": 1,
  "url": "https://www.alza.sk/...",
  "referrer": "https://www.alza.sk/",
  "accessories": null,
  "barem": null,
  "tretinka": false,
  "discountCode": null,
  "isBuyOnInstallments": false,
  "pageType": 1,
  "pageId": 18862660,
  "referrerPageType": 5,
  "referrerPageId": null,
  "addFreeAccessories": true
}
```

**Response:**
```json
{
  "d": {
    "Basket": "12,90 €",
    "Changed": true,
    "Count": 1,
    "Id": 898230944,
    "ItemsCount": 1,
    "GtmData": "[{\"data\":{\"item_id\":\"SPTgym363\",\"item_name\":\"GymBeam 100% kreatín monohydrát 250 g\",\"item_price\":\"10.8403\",\"item_price_vat\":\"12.90\",\"item_brand\":\"GymBeam\",\"item_currency\":\"EUR\",\"item_quantity\":\"1\"}}]",
    "Message": null,
    "ErrorLevel": 0
  }
}
```

**Minimal Request (len potrebné polia):**
```json
{
  "id": 7191542,
  "count": 1
}
```

---

## 4. Cart/Basket

### Get Cart Items
Získa položky v košíku.

```
GET /api/v1/anonymous/baskets/{basketId}/checkout/cart/items?country=SK
Host: www.alza.sk
```

**Response:**
```json
{
  "self": {
    "href": "...",
    "appLink": "BasketCheckoutCartItems"
  },
  "items": [
    {
      "productId": 7191542,
      "count": 1,
      "basketItemId": 898230944,
      "updateQuantityAction": {...},
      "isDelayedPayment": false
    }
  ]
}
```

### Get Cart Info
Získa info o košíku vrátane akcií.

```
GET /api/v1/visitors/{visitorId}/baskets/{basketId}/checkout/cart?country=SK
Host: www.alza.sk
```

**Response:**
```json
{
  "maxStep": 1,
  "itemsAction": {
    "href": ".../cart/items"
  },
  "emptyCartAction": {
    "name": "Vyprázdniť košík",
    "form": {
      "method": "DELETE",
      "href": ".../cart/items"
    }
  },
  "priceAction": {
    "href": ".../cart/price"
  }
}
```

### Empty Cart
Vyprázdni košík.

```
DELETE /api/v1/anonymous/baskets/{basketId}/checkout/cart/items?country=SK
Host: www.alza.sk
```

### Basket Preview
Náhľad košíka (popup).

```
GET /api/basket/{basketId}/preview
Host: www.alza.sk
```

---

## 5. Product

### Product Detail (Legacy)
Detail produktu vrátane ceny, parametrov, variantov a `descPageUrl`.

```
GET /api/router/legacy/catalog/product/{productId}?country=SK&electronicContentOnly=False
Host: www.alza.sk
```

**Poznámka:** `descPageUrl` obsahuje HTML stránku s popisom produktu (mobilná verzia).

### Product Availability
Dostupnosť produktu.

```
GET /api/productAvailability/v1/anonymous/products/{productId}?country=SK
Host: www.alza.sk
```

**Response:**
```json
{
  "type": 0,
  "title": "Na sklade > 50 ks",
  "description": "Na sklade > 50 ks",
  "expectedStockDate": null
}
```

### Product Reviews
Štatistiky recenzií.

```
GET /api/catalog/v2/commodities/{productId}/reviewStats?country=SK
Host: webapi.alza.cz
```

### Check Service Region
Kontrola dostupnosti služieb v regióne.

```
POST /api/ProductFull/IsCommodityWithServiceLimitedToRegion
Host: www.alza.sk
Content-Type: application/json
```

**Request:**
```json
{
  "commodityId": 7191542
}
```

---

## 6. Categories

### Top Producers
Top výrobcovia v kategórii.

```
GET /api/category/v1/categories/{categoryId}/topProducers?country=SK
Host: webapi.alza.cz
```

### Promo Sections
Promo sekcie na homepage.

```
GET /api/catalog/v1/homePage/categories/{categoryId}/promoSections?country=SK
Host: www.alza.sk
```

### Category Products
Produkty v kategórii (cez REST service).

```
GET /Services/RestService.svc/v1/category/{categoryId}?T=CATEGORY
Host: www.alza.sk
```

---

## 7. Authentication

### OAuth/OIDC Flow
Alza používa OAuth 2.0 s OIDC cez `identity.alza.cz`.

**Flow:**
1. `GET /external/login?state={visitorId}` (www.alza.sk)
2. → Redirect na `identity.alza.cz/connect/authorize`
3. → Login form na `identity.alza.cz/Account/Login`
4. → POST callback s `code` + `id_token`
5. → Session cookie set

**OAuth Parameters:**
```
client_id=alza
response_type=code id_token
scope=email openid profile alza offline_access
redirect_uri=https://www.alza.sk/external/callback
response_mode=form_post
```

**⚠️ Cloudflare CAPTCHA** - Login stránka má CAPTCHA, headless prihlásenie je problematické.

---

## 8. Favorites (Commodity List)

### Add to Favorites
Pridanie do obľúbených (zistené z cart response).

```
POST /api/v1/visitors/{visitorId}/commodityList/items
Host: www.alza.sk
Content-Type: application/json
```

**Request:**
```json
{
  "items": {
    "7191542": 1
  },
  "listType": 14,
  "country": "SK"
}
```

---

## 9. Product ID Format

- **Product ID:** Číslo z URL, napr. `d7191542` → `7191542`
- **Category ID:** Číslo z URL, napr. `/kreatin/18862660.htm` → `18862660`
- **Order Code:** Textový kód, napr. `SPTgym363`
- **Visitor ID:** UUID cookie
- **Basket ID:** Číslo z API response

---

## 10. Headers

### Potrebné headers pre API volania

```http
Accept: application/json, text/plain, */*
Accept-Language: sk-SK,sk;q=0.9
Content-Type: application/json
Origin: https://www.alza.sk
Referer: https://www.alza.sk/
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36
```

### Cookies
- Session cookies z browseru
- `visitorId` cookie obsahuje visitor UUID

---

## Implementation Notes

### Playwright Session Approach

```typescript
// 1. Launch persistent browser
const browser = await chromium.launchPersistentContext('./alza-profile', {
  headless: false // potrebné pre CAPTCHA
});

// 2. Manual login (user musí vyriešiť CAPTCHA)
// 3. Extract cookies
const cookies = await context.cookies();

// 4. Use cookies for subsequent API calls
// ... alebo pokračovať s Playwright pre všetky akcie
```

### Rate Limiting
- Nie je zdokumentované
- Odporúčané: 1-2 req/sec
- Cloudflare môže blokovať pri podozrivej aktivite

---

## 9. User APIs (Authenticated)

**Poznámka:** Pre tieto endpointy potrebuješ byť prihlásený (ApplicationCookie).

### User Info
```
GET /api/users/{userId}/v2/userInfo
Host: webapi.alza.cz
```

**Response:**
```json
{
  "name": "Example User",
  "email": "user@example.com",
  "login": "login@example.com",
  "avatarUrl": "https://image.alza.cz/Images/avatars/list/06.jpg",
  "icons": [
    {"name": "Alza Plus", "href": "..."}
  ]
}
```

### User Status Summary
```
GET /api/users/{userId}/statusSummary
Host: www.alza.sk
```

**Response:**
```json
{
  "notificationsCount": 9,
  "basketProductsCount": 1,
  "basketPreviewAction": {
    "href": "https://www.alza.sk/api/basket/{basketId}/preview"
  }
}
```

---

## 10. Orders API

### List Orders
```
GET /api/users/{userId}/v1/orders
Host: www.alza.sk
```

### Active Orders
```
GET /api/users/{userId}/v1/orders/active
Host: www.alza.sk
```

### Archive Orders (with pagination)
```
GET /api/users/{userId}/v1/orders/archive?offset=0&limit=10&hideCancelledOrders=false
Host: www.alza.sk
```

**Response:**
```json
{
  "paging": {
    "limit": 10,
    "size": 217,
    "next": {"href": "...?offset=10"}
  },
  "value": [
    {
      "self": {
        "webLink": "/my-account/order-details-575172045.htm",
        "href": "/api/users/{userId}/v1/orders/575172045/"
      },
      "items": [
        {
          "commodityId": 5588044,
          "commodityName": "Proteín PROM-IN Essential CFM...",
          "imageUrl": "https://image.alza.cz/...",
          "count": 1,
          "status": "Vybavené"
        }
      ],
      "totalPrice": "66,06 €",
      "state": "Vybavená",
      "documents": [
        {"id": "5416422846", "name": "Faktúra 5416422846"}
      ]
    }
  ]
}
```

### Order Detail
```
GET /api/users/{userId}/v1/orders/{orderId}/
Host: www.alza.sk
```

---

## 11. Key IDs Reference

| ID Type | Format | Example |
|---------|--------|---------|
| User ID | number | `3434016` |
| Visitor ID | UUID | `369e419f-01d3-4d0f-b8a4-eace5aabdd40` |
| Basket ID | number | `1538710316` |
| Product ID | number | `7191542` |
| Order ID | number | `575172045` |
| Category ID | number | `18862660` |

---

## 12. Commodity Lists API (Favorites, Custom Lists)

Alza má systém "zoznamov" - vlastne viaceré košíky/wishlists.

### List Types
| Type | Popis |
|------|-------|
| `0` | Custom list (user-created) |
| `1` | Obľúbené (Favorites) |
| `9` | Často nakupujem (Frequent Purchases) |
| `14` | Uložené na neskôr (Buy Later) |
| `15` | PC Konfigurátor |

### Get All Lists
```
GET /services/restservice.svc/v1/getCommodityLists
Host: www.alza.sk
```

**Response:**
```json
{
  "data_cnt": 8,
  "data": [
    {
      "id": 44367457,
      "name": "Obľúbené",
      "itemCount": 18,
      "type": 1,
      "shareUrlCanModify": "https://www.alza.sk/nakupni-seznamy.htm?sharelist=...",
      "shareUrlCannotModify": "https://www.alza.sk/nakupni-seznamy.htm?sharelist=...",
      "canModify": true,
      "isShared": false
    }
  ],
  "user_name": "Example User",
  "user_id": 123456
}
```

### Get List Items
```
GET /services/restservice.svc/v1/getCommodityLists/{listId}
Host: www.alza.sk
```

**Response:**
```json
{
  "data": [{
    "id": 44367457,
    "name": "Obľúbené",
    "itemCount": 18,
    "items": [
      {
        "navigationUrl": "https://www.alza.sk/...",
        "priceInfoV2": {"priceWithVat": "209,90 €"},
        "count": 1,
        "onChangeItemCountClick": {
          "form": {
            "method": "POST",
            "href": "/api/basket/v1/commodityLists/{listId}/items/{itemId}"
          }
        }
      }
    ]
  }]
}
```

### Update Item in List
```
POST /api/basket/v1/commodityLists/{listId}/items/{itemId}
Host: www.alza.sk
Content-Type: application/json
```

**Body:**
```json
{"count": 2}
```

### Remove Item from List
```
POST /services/restservice.svc/v1/deleteCommodityFromList
Host: www.alza.sk
Content-Type: application/json
```

**Body:**
```json
{"id": 152772164, "productId": 7646157}
```

### Add Items to List
```
POST /api/v1/users/{userId}/commodityList/items
Host: www.alza.sk
Content-Type: application/json
```

**Body:**
```json
{
  "items": {"7191542": 1},
  "listType": 14,
  "country": "SK"
}
```

### View List in Browser
```
https://www.alza.sk/Order1.htm?listId={listId}
```

---

## 13. Frequent Purchases API

Produkty ktoré user často kupuje - ideálne pre "reorder" funkciu.

```
GET /api/users/{userId}/carousels/v1/frequentPurchases?country=SK
Host: www.alza.sk
```

**Response:**
```json
{
  "title": "Často nakupujem",
  "items": [
    {
      "id": 7816725,
      "name": "Sizeandsymmetry Creatine Creapure 500 g",
      "imageUrl": "https://image.alza.cz/...",
      "priceInfoV2": {"priceWithVat": "27,90 €"}
    }
  ]
}
```

---

## 14. Add to Favorites (TESTED & WORKING)

### Add Product to List by Type
```
POST /api/v1/users/{userId}/commodityList/items
Host: www.alza.sk
Content-Type: application/json
```

**Body:**
```json
{
  "items": {"7191542": 1},
  "listType": 1,
  "country": "SK"
}
```

**List Types:**
| listType | Popis |
|----------|-------|
| `1` | Obľúbené (Favorites) |
| `14` | Uložené na neskôr (Buy Later) |

**Response:**
```json
{"IsSuccess": true, "ErrorMessage": null}
```

**Note:** Toto pridá produkt do default zoznamu daného typu. Pre pridanie do konkrétneho custom listu (napr. "Panvice") potrebuješ iný endpoint.

---

## 15. Cart Item Update/Remove (TESTED & WORKING)

### Update/Remove Cart Item
```
POST /Services/EShopService.svc/OrderUpdate?country=SK
Host: www.alza.sk
Content-Type: application/json
```

**Request Body:**
```json
{
  "id": "898234016",
  "count": 0,
  "addHook": null,
  "source": 4,
  "accessoryvariant": null
}
```

**Notes:**
- `id` is the **basketItemId** (not productId!)
- `count: 0` removes the item
- `count: N` updates quantity to N
- `source: 4` = web cart page

---

## 16. Fast Order / Quick Buy API (TESTED & WORKING)

Rýchle objednanie produktu priamo z detailu - "Kúp na klik".

### Step 1: FastOrderSave (príprava)
```
POST /Services/EShopService.svc/FastOrderSave
Host: www.alza.sk
Content-Type: application/json
```

### Step 2: FastOrderSend (odoslanie)
```
POST /Services/EShopService.svc/FastOrderSend
Host: www.alza.sk
Content-Type: application/json
```

**Request Body (obe volania):**
```json
{
  "options": {
    "Items": [{"CommodityId": 7816725, "Count": 1}],
    "AlzaBoxId": 1009905,
    "DeliveryId": 2680,
    "PaymentId": "216",
    "PrefferedCard": "5753152",
    "IsAlzaPlus": true,
    "TotalPriceDec": 27.9,
    "IsLoggedIn": true,
    "Source": "Unknown",
    "IsDelayedPayment": false,
    "wasDeliveryPaymentChanged": true,
    "ShowAlert": false,
    "DeliveryAddressId": -1,
    "IsAddressRequired": false,
    "IsTretinka": false,
    "IsVirtual": false,
    "NeedAddress": false,
    "ShowPaymentCards": true,
    "IsBusinessCardSelected": false,
    "AddressId": -1,
    "PromoCodes": [],
    "selectedPayment": null,
    "Step": null,
    "IsDialogVisible": false,
    "SendCallback": null,
    "Note": null,
    "SelectedPayment": null,
    "AlzaPremium": false
  }
}
```

### Step 3: Payment (Adyen recurrent)
```
POST /api/payment/v3/recurrent
Host: www.alza.sk
Content-Type: application/json
```

**Request Body:**
```json
{
  "browser": {
    "screenWidth": 1800,
    "screenHeight": 1169,
    "colorDepth": 30,
    "userAgent": "Mozilla/5.0...",
    "timeZoneOffset": -60,
    "language": "sk-SK",
    "javaEnabled": false,
    "deviceFingerprint": "uuid"
  },
  "cardId": 5753152,
  "fastOrder": true,
  "orderId": "578794375",
  "afterOrderPaymentId": 2141768152,
  "deviceFingerprint": "uuid"
}
```

### Step 4: Success confirmation
```
GET /Services/EShopService.svc/FastOrderSuccessDialog?isPaid=True&masterOrderId=578794375
```

### Kľúčové IDs

| Parameter | Hodnota | Popis |
|-----------|---------|-------|
| AlzaBoxId | 1009905 | Žilina - Obvodová (Tesco) |
| DeliveryId | 2680 | AlzaBox delivery type |
| PaymentId | 216 | Kartou online |
| PrefferedCard | 5753152 | ID uloženej karty |

### Poznámky
- Vyžaduje uloženú platobnú kartu
- Vyžaduje AlzaPlus+ pre dopravu zadarmo
- `afterOrderPaymentId` sa získa z FastOrderSend response
- Objednávka sa vytvorí OKAMŽITE - bez ďalšieho potvrdenia!
