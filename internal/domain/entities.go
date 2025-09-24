package domain

type Delivery struct {
	Name    string `json:"name" validate:"required"`
	Phone   string `json:"phone" validate:"required,e164"`
	Zip     string `json:"zip" validate:"required,len=7"`
	City    string `json:"city" validate:"required"`
	Address string `json:"address" validate:"required"`
	Region  string `json:"region" validate:"required"`
	Email   string `json:"email" validate:"required,email"`
}

type Items struct {
	ChrtID     int    `json:"chrt_id" validate:"required"`
	Price      int    `json:"price" validate:"required,min=0"`
	Rid        string `json:"rid" validate:"required"`
	Name       string `json:"name" validate:"required"`
	Sale       int    `json:"sale" validate:"min=0,max=100"` // Assuming sale is a percentage
	Size       string `json:"size"`
	TotalPrice int    `json:"total_price" validate:"required,min=0"`
	NmID       int    `json:"nm_id" validate:"required"`
	Brand      string `json:"brand" validate:"required"`
}

type Order struct {
	OrderUID          string   `json:"order_uid"`
	Entry             string   `json:"entry" validate:"required"`
	InternalSignature string   `json:"internal_signature"`
	Payment           Payment  `json:"payment" validate:"required"`
	Items             []Items  `json:"items" validate:"required,dive,required"` // Dive into the slice and validate each item
	Locale            string   `json:"locale" validate:"required"`
	CustomerID        string   `json:"customer_id" validate:"required"`
	TrackNumber       string   `json:"track_number" validate:"required"`
	DeliveryService   string   `json:"delivery_service" validate:"required"`
	Shardkey          string   `json:"shardkey" validate:"required"`
	SmID              int      `json:"sm_id" validate:"required"`
	Delivery          Delivery `json:"delivery" validate:"required"` // Add validation for Delivery struct
}

type OrderOut struct {
	OrderUID        string `json:"order_uid"`
	Entry           string `json:"entry"`
	TotalPrice      int    `json:"total_price"`
	CustomerID      string `json:"customer_id"`
	TrackNumber     string `json:"track_number"`
	DeliveryService string `json:"delivery_service"`
}

type Payment struct {
	Transaction  string `json:"transaction" validate:"required"`
	Currency     string `json:"currency" validate:"required,len=3"` // Assuming currency is a 3-letter code
	Provider     string `json:"provider" validate:"required"`
	Amount       int    `json:"amount" validate:"required,min=0"`
	PaymentDt    int    `json:"payment_dt" validate:"required"`
	Bank         string `json:"bank" validate:"required"`
	DeliveryCost int    `json:"delivery_cost" validate:"required,min=0"`
	GoodsTotal   int    `json:"goods_total" validate:"required,min=0"`
}
