package order_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	logger "GuiFernandess7/gothenticate/internal/infra/log"
	customvalidator "GuiFernandess7/gothenticate/internal/infra/validator"
	"GuiFernandess7/gothenticate/internal/middleware"
	order "GuiFernandess7/gothenticate/internal/modules/order"
	domain "GuiFernandess7/gothenticate/internal/modules/order/domain"

	"github.com/labstack/echo/v4"
)

type noopLogger struct{}

func (noopLogger) Info(string)           {}
func (noopLogger) Warn(string)           {}
func (noopLogger) Error(string)          {}
func (noopLogger) Infof(string, ...any)  {}
func (noopLogger) Warnf(string, ...any)  {}
func (noopLogger) Errorf(string, ...any) {}

var _ logger.Logger = noopLogger{}

func newJSONContext(method, target string, body []byte) (*httptest.ResponseRecorder, echo.Context) {
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e := echo.New()
	e.Validator = customvalidator.New() // mirrors the registration done in server.go
	return rec, e.NewContext(req, rec)
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("failed to decode response body %q: %v", rec.Body.String(), err)
	}
	return out
}

// ---------- GetOrders ----------

// Regression guard: GetOrders calls c.Validate(&filters) unconditionally.
// Without a registered echo.Validator (see server.go's e.Validator = ...),
// c.Validate returns echo.ErrValidatorNotRegistered, which the handler now
// turns into a proper 400 JSON response instead of leaking a bare Go error
// (which is what used to happen before this was fixed).
func TestOrderHandler_GetOrders_RespondsWith400IfNoValidatorRegistered(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	rec := httptest.NewRecorder()
	e := echo.New() // deliberately NOT setting e.Validator
	c := e.NewContext(req, rec)

	h := &order.OrderHandler{Log: noopLogger{}}
	if err := h.GetOrders(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestOrderHandler_GetOrders_InvalidFilters(t *testing.T) {
	rec, c := newJSONContext(http.MethodGet, "/orders?limit=0&page=-1", nil)
	h := &order.OrderHandler{Log: noopLogger{}}

	if err := h.GetOrders(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestOrderHandler_GetOrders_InvalidQueryParams(t *testing.T) {
	rec, c := newJSONContext(http.MethodGet, "/orders?page=not-a-number", nil)
	h := &order.OrderHandler{Log: noopLogger{}}

	if err := h.GetOrders(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if body := decodeBody(t, rec); body["message"] != "invalid payload" {
		t.Fatalf("unexpected message: %v", body["message"])
	}
}

// ---------- AddOrder ----------

func TestOrderHandler_AddOrder_InvalidPayload(t *testing.T) {
	rec, c := newJSONContext(http.MethodPost, "/orders", []byte("{invalid-json"))
	h := &order.OrderHandler{Log: noopLogger{}}

	if err := h.AddOrder(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if body := decodeBody(t, rec); body["message"] != "invalid payload" {
		t.Fatalf("unexpected message: %v", body["message"])
	}
}

func TestOrderHandler_AddOrder_FailsValidation(t *testing.T) {
	// TotalAmount is zero -- validate:"required,gt=0" on AddOrderRequest
	// should reject this before the handler even checks authentication.
	payload, _ := json.Marshal(order.AddOrderRequest{
		TotalAmount: 0,
		Weight:      2,
		Zip:         "01310-100",
		Carrier:     "standard_mail",
	})
	rec, c := newJSONContext(http.MethodPost, "/orders", payload)
	h := &order.OrderHandler{Log: noopLogger{}}

	if err := h.AddOrder(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestOrderHandler_AddOrder_MissingAuth(t *testing.T) {
	payload, _ := json.Marshal(order.AddOrderRequest{
		TotalAmount: 100,
		Weight:      2,
		Zip:         "01310-100",
		Carrier:     "standard_mail",
	})
	rec, c := newJSONContext(http.MethodPost, "/orders", payload)
	// Deliberately NOT calling c.Set(middleware.ContextKeyUserID, ...) here,
	// simulating a request that somehow reached the handler without the
	// JWTAuth middleware having run.
	h := &order.OrderHandler{Log: noopLogger{}}

	if err := h.AddOrder(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	if body := decodeBody(t, rec); body["message"] != "missing or invalid authentication" {
		t.Fatalf("unexpected message: %v", body["message"])
	}
}

func TestOrderHandler_AddOrder_UnknownCarrier(t *testing.T) {
	payload, _ := json.Marshal(order.AddOrderRequest{
		TotalAmount: 100,
		Weight:      2,
		Zip:         "01310-100",
		Carrier:     "teleport",
	})
	rec, c := newJSONContext(http.MethodPost, "/orders", payload)
	c.Set(middleware.ContextKeyUserID, int64(1))
	h := &order.OrderHandler{Log: noopLogger{}}

	if err := h.AddOrder(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestOrderHandler_AddOrder_DroneWeightLimitExceeded(t *testing.T) {
	payload, _ := json.Marshal(order.AddOrderRequest{
		TotalAmount: 100,
		Weight:      6,
		Zip:         "01310-100",
		Carrier:     "local_drone",
	})
	rec, c := newJSONContext(http.MethodPost, "/orders", payload)
	c.Set(middleware.ContextKeyUserID, int64(1))
	h := &order.OrderHandler{Log: noopLogger{}}

	if err := h.AddOrder(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestOrderHandler_AddOrder_Success(t *testing.T) {
	payload, _ := json.Marshal(order.AddOrderRequest{
		TotalAmount: 250,
		Weight:      2,
		Zip:         "01310-100",
		Carrier:     "standard_mail",
	})
	rec, c := newJSONContext(http.MethodPost, "/orders", payload)
	c.Set(middleware.ContextKeyUserID, int64(42))
	db := newTestDB(t)
	h := &order.OrderHandler{Repo: *order.NewOrderRepository(db), Log: noopLogger{}}

	if err := h.AddOrder(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := decodeBody(t, rec)
	data, ok := body["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected a \"data\" object in the response, got: %v", body)
	}
	if data["status"] != "pending" {
		t.Fatalf("expected status \"pending\", got: %v", data["status"])
	}
	if int64(data["user_id"].(float64)) != 42 {
		t.Fatalf("expected user_id 42 (from context, not the request body), got: %v", data["user_id"])
	}
	id, ok := data["id"].(float64)
	if !ok || id == 0 {
		t.Fatalf("expected a non-zero id for the persisted order, got: %v", data["id"])
	}
	wantShippingCost := (domain.BaseRate + (2 * 1.5)) * 0.9 // standard_mail, 10% discount: zip starts with '0'
	if data["shipping_cost"].(float64) != wantShippingCost {
		t.Fatalf("expected shipping_cost %v, got %v", wantShippingCost, data["shipping_cost"])
	}

	persisted, err := order.NewOrderRepository(db).GetOrderByID(context.Background(), int64(id))
	if err != nil {
		t.Fatalf("expected the order to actually be persisted, got: %v", err)
	}
	if persisted.UserID != 42 || persisted.Status != "pending" || persisted.TotalAmount != 250 {
		t.Fatalf("persisted order does not match the request: %+v", persisted)
	}
}

// ---------- Stubs (document current placeholder behavior) ----------

// GetOrderByID and UpdateOrderStatus are NOT covered here anymore: both now
// have real logic (they used to be plain stubs), but GetOrderByID's
// repository method still has known bugs (queries a nonexistent "email"
// column, and Scan() only reads 1 of the row's 8 columns) that are tracked
// separately and not yet fixed, and UpdateOrderStatus calls GetOrderByID
// internally. Exercising either handler today would just panic on a nil
// *sql.DB, since OrderHandler.Repo has no real connection in these tests.

// ---------- DeleteOrder ----------
//
// DeleteOrder wires a real OrderRepository backed by the in-memory sqlite DB
// (same helpers as repository_test.go) so the full id-parse -> lookup ->
// in-progress-guard -> delete flow is exercised, not just the early
// bad-path-parameter branch.

func TestOrderHandler_DeleteOrder_InvalidID(t *testing.T) {
	rec, c := newJSONContext(http.MethodDelete, "/orders/abc", nil)
	c.SetParamNames("id")
	c.SetParamValues("abc")
	h := &order.OrderHandler{Log: noopLogger{}}

	if err := h.DeleteOrder(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if body := decodeBody(t, rec); body["message"] != "something went wrong." {
		t.Fatalf("unexpected message: %v", body["message"])
	}
}

func TestOrderHandler_DeleteOrder_NotFound(t *testing.T) {
	db := newTestDB(t)
	rec, c := newJSONContext(http.MethodDelete, "/orders/999", nil)
	c.SetParamNames("id")
	c.SetParamValues("999")
	h := &order.OrderHandler{Repo: *order.NewOrderRepository(db), Log: noopLogger{}}

	if err := h.DeleteOrder(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
	if body := decodeBody(t, rec); body["message"] != "not found." {
		t.Fatalf("unexpected message: %v", body["message"])
	}
}

func TestOrderHandler_DeleteOrder_RejectsInProgressOrders(t *testing.T) {
	for _, status := range []string{"preparing", "sent"} {
		t.Run(status, func(t *testing.T) {
			db := newTestDB(t)
			id := seedOrder(t, db, 1, status, "standard_mail", 10, time.Now())

			rec, c := newJSONContext(http.MethodDelete, "/orders/"+strconv.FormatInt(id, 10), nil)
			c.SetParamNames("id")
			c.SetParamValues(strconv.FormatInt(id, 10))
			h := &order.OrderHandler{Repo: *order.NewOrderRepository(db), Log: noopLogger{}}

			if err := h.DeleteOrder(c); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
			if body := decodeBody(t, rec); body["message"] != "cannot delete order in progress." {
				t.Fatalf("unexpected message: %v", body["message"])
			}

			if _, err := order.NewOrderRepository(db).GetOrderByID(context.Background(), id); err != nil {
				t.Fatalf("expected order %d to still exist after a rejected delete, got: %v", id, err)
			}
		})
	}
}

func TestOrderHandler_DeleteOrder_Success(t *testing.T) {
	db := newTestDB(t)
	id := seedOrder(t, db, 1, "pending", "standard_mail", 10, time.Now())

	rec, c := newJSONContext(http.MethodDelete, "/orders/"+strconv.FormatInt(id, 10), nil)
	c.SetParamNames("id")
	c.SetParamValues(strconv.FormatInt(id, 10))
	h := &order.OrderHandler{Repo: *order.NewOrderRepository(db), Log: noopLogger{}}

	if err := h.DeleteOrder(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// DeleteOrder returns nil directly on success instead of calling
	// c.JSON/c.NoContent, so nothing is ever written to the response: the
	// recorder keeps its default 200 status and an empty body.
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected an empty body since the handler never writes a response, got: %q", rec.Body.String())
	}

	if _, err := order.NewOrderRepository(db).GetOrderByID(context.Background(), id); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected order %d to be deleted, got err=%v", id, err)
	}
}
