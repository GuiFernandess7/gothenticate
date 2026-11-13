package order

import (
	"time"

	domain "GuiFernandess7/gothenticate/internal/modules/order/domain"
)

type UpdateOrderStatusRequest struct {
	Status string `json:"status"`
}

// Order is an alias for domain.Order: the order module's handler/repository
// layer depends on domain, so the Order type has to live there to avoid an
// import cycle, but everything in this package keeps referring to it as
// Order.
type Order = domain.Order

type OrderFilters struct {
    Page   int    `query:"page" validate:"omitempty,gte=1"`
    Limit  int    `query:"limit" validate:"omitempty,gte=1,lte=100"`
    Search string `query:"search"`
    Status string `query:"status"`
}

type AddOrderRequest struct {
    TotalAmount float64 `json:"total_amount" validate:"required,gt=0"`
    Weight      float64 `json:"weight" validate:"required,gt=0"`
    Zip         string  `json:"zip" validate:"required"`
    Carrier     string  `json:"carrier" validate:"required"`
}

type AddOrderResponse struct {
	ID           int64   `json:"id" validate:"required"`
	UserID       int64   `json:"user_id" validate:"required"`
    Status       string  `json:"status" validate:"required"`
    TotalAmount  float64 `json:"total_amount" validate:"required,gt=0"`
    Weight       float64 `json:"weight" validate:"required,gt=0"`
    Zip          string  `json:"zip" validate:"required"`
    Carrier      string  `json:"carrier" validate:"required"`
	ShippingCost float64 `json:"shipping_cost" validate:"required"`
}

type UpdateOrderRequest struct {
    OrderID  int64    `json:"order_id"`
    Action   string   `json:"action"`
}

type GetOrderByIDRequest struct {
    ID    int64  `json:"id"`
}

type OrderResponse struct {
    Status      string   `json:"status"`
	TotalAmount float64  `json:"total"`
	Weight      float64  `json:"weight"`
	Zip         string   `json:"zip"`
	Carrier     string   `json:"carrier"`
	CreatedAt   time.Time `json:"created_at"`
}

type OrdersResponse struct {
    Orders []Order `json:"orders"`
    Total  int     `json:"total"`
    Page   int     `json:"page"`
}

