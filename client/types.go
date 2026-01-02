package client

type UserStatusResponse struct {
	UserID    int    `json:"userId"`
	BasketID  int    `json:"basketId"`
	UserName  string `json:"userName"`
	BasketCnt int    `json:"basketItemsCount"`
	OrdersCnt int    `json:"ordersCount"`
	FavCnt    int    `json:"watchDogCommoditiesCount"`
	IsPremium bool   `json:"isAlzaPlus"`
}

type SearchResult struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	Code         string  `json:"code"`
	Price        float64 `json:"price"`
	PriceStr     string  `json:"priceStr"`
	Availability string  `json:"availability"`
	ImageURL     string  `json:"imageUrl"`
	URL          string  `json:"url"`
}

type WhisperResponse struct {
	Items []struct {
		ItemID    int     `json:"itemId"`
		Name      string  `json:"name"`
		ItemCode  string  `json:"itemCode"`
		Price     string  `json:"priceWithVat"`
		PriceFull float64 `json:"priceWithVatValue"`
		Img       string  `json:"imgUrl"`
		URL       string  `json:"url"`
		Avail     int     `json:"availability"`
	} `json:"items"`
}

type CommodityList struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	ItemCount int    `json:"itemCount"`
	Type      int    `json:"type"`
	CanModify bool   `json:"canModify"`
}

type ListItem struct {
	NavigationURL string `json:"navigationUrl"`
	Count         int    `json:"count"`
	Name          string `json:"-"`
	Price         string `json:"-"`
}

type AddToListResponse struct {
	IsSuccess    bool   `json:"IsSuccess"`
	ErrorMessage string `json:"ErrorMessage"`
}

type CartItem struct {
	ProductID    int    `json:"productId"`
	Count        int    `json:"count"`
	BasketItemID int    `json:"basketItemId"`
	Name         string `json:"name"`
	Price        string `json:"price"`
	ImageURL     string `json:"imageUrl"`
	URL          string `json:"url"`
}

type Order struct {
	ID         string `json:"orderId"`
	Date       string `json:"orderDate"`
	Status     string `json:"status"`
	TotalPrice string `json:"totalPrice"`
}

type ProductDetail struct {
	ID                  int                     `json:"id"`
	Name                string                  `json:"name"`
	Price               string                  `json:"price"`
	PriceWithoutVat     string                  `json:"priceWithoutVat,omitempty"`
	PriceNoCurrency     float64                 `json:"priceNoCurrency,omitempty"`
	DiscountPercent     *int                    `json:"discountPercent,omitempty"`
	CashBackPriceLabel  string                  `json:"cashBackPriceLabel,omitempty"`
	CashBackPrice       string                  `json:"cashBackPrice,omitempty"`
	DiscountDescription string                  `json:"discountDescription,omitempty"`
	Availability        string                  `json:"availability,omitempty"`
	AvailabilityDetail  string                  `json:"availabilityDetail,omitempty"`
	ExpectedStockDate   string                  `json:"expectedStockDate,omitempty"`
	Description         string                  `json:"description,omitempty"`
	Parameters          []ProductParameterGroup `json:"parameters,omitempty"`
	Variants            []ProductVariant        `json:"variants,omitempty"`
	PromoPrices         []ProductPromoPrice     `json:"promoPrices,omitempty"`
}

type ProductParameterGroup struct {
	Name       string             `json:"name"`
	Parameters []ProductParameter `json:"parameters"`
}

type ProductParameter struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

type ProductVariant struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	ImageURL   string `json:"imageUrl,omitempty"`
	IsSelected bool   `json:"isSelected"`
}

type ProductPromoPrice struct {
	Name             string  `json:"name"`
	Price            string  `json:"price"`
	Code             string  `json:"code,omitempty"`
	UnformattedPrice float64 `json:"unformattedPrice,omitempty"`
}
