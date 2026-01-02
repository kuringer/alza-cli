package client

const (
	EndpointAccessTokenPath        = "/api/identity/v1/accesstoken"
	EndpointCommodityLists         = "/services/restservice.svc/v1/getCommodityLists"
	EndpointCommodityListItems     = "/services/restservice.svc/v1/getCommodityLists/%d"
	EndpointCommodityListCreate    = "/services/restservice.svc/v1/createCommodityList"
	EndpointCommodityListDelete    = "/services/restservice.svc/v1/deleteCommodityFromList"
	EndpointCommodityListAddItem   = "/Services/EShopService.svc/AddCommodityToShoppingList"
	EndpointUserCommodityListItems = "/api/v1/users/%s/commodityList/items"

	EndpointUserStatusSummary = "/api/users/%s/statusSummary"

	EndpointCartItems   = "/api/v1/anonymous/baskets/%s/checkout/cart/items?country=SK"
	EndpointCartPreview = "/api/basket/%s/preview"

	EndpointOrderCommodity = "/Services/EShopService.svc/OrderCommodity"
	EndpointOrderUpdate    = "/Services/EShopService.svc/OrderUpdate?country=SK"

	EndpointSearchService = "/Services/RestService.svc/v5/search"
	EndpointWhisperAnon   = "https://webapi.alza.cz/api/anonymous/search/whisperer/v1/whisper"
	EndpointWhisperUser   = "https://webapi.alza.cz/api/users/%s/search/whisperer/v1/whisper"

	EndpointOrdersArchive = "/api/users/%s/v1/orders/archive?offset=0&limit=%d&hideCancelledOrders=false"
	EndpointOrdersActive  = "/api/users/%s/v1/orders/active"

	EndpointProductDetail           = "/api/router/legacy/catalog/product/%d?country=SK&electronicContentOnly=False"
	EndpointProductAvailabilityUser = "/api/productAvailability/v1/users/%s/products/%d?country=SK"
	EndpointProductAvailabilityAnon = "/api/productAvailability/v1/anonymous/products/%d?country=SK"

	EndpointFastOrderSave = "/Services/EShopService.svc/FastOrderSave"
	EndpointFastOrderSend = "/Services/EShopService.svc/FastOrderSend"
	EndpointPaymentRepeat = "/api/payment/v3/recurrent"
)
