package order

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	logger "GuiFernandess7/gothenticate/internal/infra/log"
	"GuiFernandess7/gothenticate/internal/middleware"
	"GuiFernandess7/gothenticate/internal/modules/order/domain"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

type OrderHandler struct {
	Repo      OrderRepository
	Log       logger.Logger
	JWTSecret []byte
	RDB       *redis.Client
}

func NewOrderFilters() OrderFilters {
    return OrderFilters{
        Page:  1,
        Limit: 10,
    }
}

func (o *OrderHandler) GetOrders(c echo.Context) error {
	// Extract filters params
	filters := NewOrderFilters()
	if err := (&echo.DefaultBinder{}).BindQueryParams(c, &filters); err != nil {
        o.Log.Errorf("Failed to bind order filters: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "invalid payload",
		})
    }

	if err := c.Validate(&filters); err != nil {
        o.Log.Errorf("Order filters failed validation: %v", err)
        return c.JSON(http.StatusBadRequest, map[string]string{
            "message": "invalid payload",
        })
    }

	// 2. Call database.
	ctx := c.Request().Context()
	userID, ok := c.Get(middleware.ContextKeyUserID).(int64)
	if !ok {
		o.Log.Warnf("GetOrders called without a valid user id in context")
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"message": "missing or invalid authentication",
		})
	}

	orders, total, page, err := o.Repo.GetOrders(ctx, filters, userID)
	if err != nil {
		o.Log.Errorf("Failed to fetch orders for user %d: %v", userID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "something went wrong.",
		})
	}

	ordersResponse := OrdersResponse{
		Orders: orders,
		Total: total,
		Page: page,
	}

	// 3. Return the paginated orders list.
	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": ordersResponse,
	})

}

func (o *OrderHandler) AddOrder(c echo.Context) error {
	// 1. Validate and extract payload information.
	var requestBody AddOrderRequest
	if c.Request().ContentLength != 0 {
		if err := c.Bind(&requestBody); err != nil {
			o.Log.Errorf("Failed to bind add order payload: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]string{
				"message": "invalid payload",
			})
		}
	}

	if err := c.Validate(&requestBody); err != nil {
		o.Log.Errorf("Add order payload failed validation: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "invalid payload",
		})
	}

	userID, ok := c.Get(middleware.ContextKeyUserID).(int64)
	if !ok {
		o.Log.Warnf("AddOrder called without a valid user id in context")
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"message": "missing or invalid authentication",
		})
	}

	// 2. Call Domain (design pattern strategy -> shipping calculation).
	orderDetails := domain.OrderDetails{
		TotalWeightKg:  requestBody.Weight,
		DestinationZip: requestBody.Zip,
	}
	shippingCost, err := domain.CalculateShipping(requestBody.Carrier, orderDetails)
	if err != nil {
		o.Log.Warnf("Failed to calculate shipping for carrier %q: %v", requestBody.Carrier, err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
	}

	// 3. Insert in database
	ctx := c.Request().Context()
	createdOrder, err := o.Repo.AddOrder(ctx, Order{
		UserID:      userID,
		Status:      "pending",
		TotalAmount: requestBody.TotalAmount,
		Weight:      requestBody.Weight,
		Zip:         requestBody.Zip,
		Carrier:     requestBody.Carrier,
	})
	if err != nil {
		o.Log.Errorf("Failed to create order for user %d: %v", userID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "something went wrong.",
		})
	}

	response := AddOrderResponse{
		ID:           createdOrder.ID,
		UserID:       createdOrder.UserID,
		Status:       createdOrder.Status,
		TotalAmount:  createdOrder.TotalAmount,
		Weight:       createdOrder.Weight,
		Zip:          createdOrder.Zip,
		Carrier:      createdOrder.Carrier,
		ShippingCost: shippingCost,
	}

	// 4. Return order payload and shipping total.
	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": response,
	})
}

func (o *OrderHandler) GetOrderByID(c echo.Context) error {
	// 1. Validate and extract payload information.
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		o.Log.Errorf("Failed to parse order id from path: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "invalid payload",
		})
	}

	ctx := c.Request().Context()
	userID, ok := c.Get(middleware.ContextKeyUserID).(int64)
	if !ok {
		o.Log.Warnf("GetOrderByID called without a valid user id in context")
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"message": "missing or invalid authentication",
		})
	}

	// 2. Find on redis.
	var order *Order
	key := fmt.Sprintf("order:%d", id)

	if o.RDB != nil {
		val, err := o.RDB.Get(ctx, key).Result()

		if err == nil { // KEY FOUND ON REDIS

			// 2.1 Deserializing body
			if err := json.Unmarshal([]byte(val), &order); err != nil {
				o.Log.Errorf("Failed to unmarshal order from redis: %v", err)
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"message": "something went wrong.",
				})
			}

			// 2.2 Verifying user
			if userID != order.UserID {
				o.Log.Warnf("User %d attempted to access order %d owned by user %d", userID, id, order.UserID)
				return c.JSON(http.StatusNotFound, map[string]string{
					"message": "not found.",
				})
			}

			// 2.3 Mount object
			orderResponse := OrderResponse{
				Status: order.Status,
				TotalAmount: order.TotalAmount,
				Weight: order.Weight,
				Zip: order.Zip,
				Carrier: order.Carrier,
				CreatedAt: order.CreatedAt,
			}

			// 2.4 Return response
			return c.JSON(http.StatusOK, map[string]interface{}{
				"data": orderResponse,
			})

		} else if errors.Is(err, redis.Nil) {
			o.Log.Infof("Order %d not found in redis, fetching from database...", id)
		} else {
			o.Log.Errorf("Redis error while fetching order %d: %v", id, err)
		}
	}

	// 3. Find on database
	order, err = o.Repo.GetOrderByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			o.Log.Warnf("Order %d not found", id)
			return c.JSON(http.StatusNotFound, map[string]string{
				"message": "not found.",
			})
		}
		o.Log.Errorf("Failed to fetch order %d: %v", id, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "something went wrong.",
		})
	}

	if userID != order.UserID {
		o.Log.Warnf("User %d attempted to access order %d owned by user %d", userID, id, order.UserID)
		return c.JSON(http.StatusNotFound, map[string]string{
			"message": "not found.",
		})
	}

	// 4. Save on redis
	jsonData, err := json.Marshal(order)
	if err != nil {
		o.Log.Warnf("error serializing redis message error: %v", err)
	}

	keyID := fmt.Sprintf("order:%d", id)
	if o.RDB != nil {
		err = o.RDB.Set(ctx, keyID, string(jsonData), 24*time.Hour).Err()
		if err != nil {
			o.Log.Warnf("Order was not saved on redis, ID: %s", keyID)
		}
	}

	// 5. Mount object
	orderResponse := OrderResponse{
		Status: order.Status,
		TotalAmount: order.TotalAmount,
		Weight: order.Weight,
		Zip: order.Zip,
		Carrier: order.Carrier,
		CreatedAt: order.CreatedAt,
	}

	// 6. Return product payload
	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": orderResponse,
	})
}


func (o *OrderHandler) UpdateOrderStatus(c echo.Context) error {
	// 1. Validate and extract payload information.
	var requestBody UpdateOrderRequest
	if c.Request().ContentLength != 0 {
		if err := c.Bind(&requestBody); err != nil {
			o.Log.Errorf("Failed to bind add order payload: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]string{
				"message": "invalid payload",
			})
		}
	}
	// 2. Call Domain (design pattern state -> order status).
	ctx := c.Request().Context()
	order, err := o.Repo.GetOrderByID(ctx, requestBody.OrderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			o.Log.Warnf("Order %d not found", requestBody.OrderID)
			return c.JSON(http.StatusNotFound, map[string]string{
				"message": "not found.",
			})
		}
		o.Log.Errorf("Failed to fetch order %d: %v", requestBody.OrderID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "something went wrong",
		})
	}

	orderStatus := domain.NewOrderStatus(
		domain.OrderStatus(order.Status),
		&domain.NotificadorEstoque{},
		&domain.NotificadorFiscal{},
		&domain.NotificadorEmail{},
	)

	switch requestBody.Action {
    case "pagar":
        err = orderStatus.Pagar(order)
    case "enviar":
        err = orderStatus.Enviar(order)
    case "cancelar":
        err = orderStatus.Cancelar(order)
    }
	if err != nil {
		o.Log.Warnf("Invalid order status transition for order %d (action %q, current status %q): %v", order.ID, requestBody.Action, order.Status, err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "invalid operation",
		})
	}

	// 2. Call database, update product.
	order, err = o.Repo.UpdateOrderStatus(ctx, order.ID, string(orderStatus.Label()))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			o.Log.Warnf("Order %d not found when updating status", order.ID)
			return c.JSON(http.StatusNotFound, map[string]string{
				"message": "not found.",
			})
		}
		o.Log.Errorf("Failed to update order %d status: %v", order.ID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "something went wrong",
		})
	}

	// 3. Delete order on redis
	key := fmt.Sprintf("order:%d", order.ID)
	if o.RDB != nil {
		deletedCount, err := o.RDB.Del(ctx, key).Result()
		if err != nil {
			o.Log.Infof("an error ocourred while deleting redis order: %v", err)
		}

		if deletedCount == 0 {
			o.Log.Infof("No order was deleted from redis")
		} else {
			o.Log.Infof("Key deleted successfully! %v", order.ID)
		}
	}

	// 3. return OK (or product body)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": order,
	})
}


func (o *OrderHandler) DeleteOrder(c echo.Context) error {
	// 1. Validate and extract payload information.
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		o.Log.Errorf("Failed to parse order id from path: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "something went wrong.",
		})
	}

	// 2. Call database, delete product.
	ctx := c.Request().Context()
	order, err := o.Repo.GetOrderByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			o.Log.Warnf("Order %d not found", id)
			return c.JSON(http.StatusNotFound, map[string]string{
				"message": "not found.",
			})
		}
		o.Log.Errorf("Failed to fetch order %d: %v", id, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "something went wrong.",
		})
	}

	if domain.OrderStatus(order.Status) == domain.StatusOrderPreparing || domain.OrderStatus(order.Status) == domain.StatusOrderSent {
		o.Log.Warnf("Rejected delete of order %d: order is in progress (status %q)", order.ID, order.Status)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "cannot delete order in progress.",
		})
	}

	err = o.Repo.DeleteOrder(ctx, order.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			o.Log.Warnf("Order %d not found when deleting", order.ID)
			return c.JSON(http.StatusNotFound, map[string]string{
				"message": "not found.",
			})
		}
		o.Log.Errorf("Failed to delete order %d: %v", order.ID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "something went wrong.",
		})
	}

	key := fmt.Sprintf("order:%d", order.ID)
	if o.RDB != nil {
		deletedCount, err := o.RDB.Del(ctx, key).Result()
		if err != nil {
			o.Log.Infof("an error ocourred while deleting redis order: %v", err)
		}

		if deletedCount == 0 {
			o.Log.Infof("No order was deleted from redis")
		} else {
			o.Log.Infof("Key deleted successfully! %v", order.ID)
		}
	}

	return nil
}
