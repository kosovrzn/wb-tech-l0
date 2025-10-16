package domain

import "time"

type Order struct {
	OrderUID          string    `json:"order_uid" validate:"required,alphanumunicode,max=64"`
	TrackNumber       string    `json:"track_number" validate:"required,printascii,max=64"`
	Entry             string    `json:"entry" validate:"required,printascii,max=32"`
	Delivery          Delivery  `json:"delivery" validate:"required"`
	Payment           Payment   `json:"payment" validate:"required"`
	Items             []Item    `json:"items" validate:"required,min=1,dive"`
	Locale            string    `json:"locale" validate:"required,alpha,len=2"`
	InternalSignature string    `json:"internal_signature" validate:"omitempty,printascii,max=128"`
	CustomerID        string    `json:"customer_id" validate:"required,printascii,max=64"`
	DeliveryService   string    `json:"delivery_service" validate:"required,printascii,max=64"`
	Shardkey          string    `json:"shardkey" validate:"required,numeric"`
	SmID              int       `json:"sm_id" validate:"gte=0"`
	DateCreated       time.Time `json:"date_created" validate:"required"`
	OofShard          string    `json:"oof_shard" validate:"required,numeric"`
}

type Delivery struct {
	Name    string `json:"name" validate:"required,printascii,max=128"`
	Phone   string `json:"phone" validate:"required,e164"`
	Zip     string `json:"zip" validate:"required,numeric"`
	City    string `json:"city" validate:"required,printascii,max=128"`
	Address string `json:"address" validate:"required,printascii,max=256"`
	Region  string `json:"region" validate:"required,printascii,max=128"`
	Email   string `json:"email" validate:"required,email"`
}

type Payment struct {
	Transaction  string `json:"transaction" validate:"required,alphanumunicode,max=64"`
	RequestID    string `json:"request_id" validate:"omitempty,printascii,max=64"`
	Currency     string `json:"currency" validate:"required,uppercase,len=3"`
	Provider     string `json:"provider" validate:"required,printascii,max=64"`
	Amount       int    `json:"amount" validate:"gt=0"`
	PaymentDT    int64  `json:"payment_dt" validate:"gt=0"`
	Bank         string `json:"bank" validate:"required,printascii,max=64"`
	DeliveryCost int    `json:"delivery_cost" validate:"gte=0"`
	GoodsTotal   int    `json:"goods_total" validate:"gte=0"`
	CustomFee    int    `json:"custom_fee" validate:"gte=0"`
}

type Item struct {
	ChrtID      int64  `json:"chrt_id" validate:"gt=0"`
	TrackNumber string `json:"track_number" validate:"required,printascii,max=64"`
	Price       int    `json:"price" validate:"gte=0"`
	RID         string `json:"rid" validate:"required,printascii,max=64"`
	Name        string `json:"name" validate:"required,printascii,max=128"`
	Sale        int    `json:"sale" validate:"gte=0"`
	Size        string `json:"size" validate:"required,printascii,max=32"`
	TotalPrice  int    `json:"total_price" validate:"gte=0"`
	NmID        int64  `json:"nm_id" validate:"gt=0"`
	Brand       string `json:"brand" validate:"required,printascii,max=128"`
	Status      int    `json:"status" validate:"gte=0"`
}
