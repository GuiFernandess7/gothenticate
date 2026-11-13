package order_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	order "GuiFernandess7/gothenticate/internal/modules/order"

	_ "modernc.org/sqlite"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`
		CREATE TABLE orders (
		    id           INTEGER PRIMARY KEY AUTOINCREMENT,
		    user_id      INTEGER NOT NULL,
		    status       TEXT NOT NULL DEFAULT 'pending',
		    total_amount REAL NOT NULL,
		    weight       REAL NOT NULL DEFAULT 0,
		    zip          TEXT NOT NULL,
		    carrier      TEXT NOT NULL,
		    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("failed to create orders table: %v", err)
	}

	return db
}

func seedOrder(t *testing.T, db *sql.DB, userID int64, status, carrier string, total float64, createdAt time.Time) int64 {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO orders (user_id, status, total_amount, weight, zip, carrier, created_at) VALUES (?, ?, ?, 1, '00000', ?, ?)`,
		userID, status, total, carrier, createdAt,
	)
	if err != nil {
		t.Fatalf("failed to seed order: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read last insert id: %v", err)
	}
	return id
}

func TestOrderRepository_GetOrders_Pagination(t *testing.T) {
	db := newTestDB(t)
	repo := order.NewOrderRepository(db)

	base := time.Now()
	for i := 0; i < 15; i++ {
		seedOrder(t, db, 1, "pending", "standard_mail", 10, base.Add(time.Duration(i)*time.Minute))
	}

	orders, total, page, err := repo.GetOrders(context.Background(), order.OrderFilters{Page: 2, Limit: 10}, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 15 {
		t.Fatalf("expected total 15, got %d", total)
	}
	if page != 2 {
		t.Fatalf("expected page 2, got %d", page)
	}
	if len(orders) != 5 {
		t.Fatalf("expected 5 orders on page 2 (15 total, limit 10), got %d", len(orders))
	}
}

func TestOrderRepository_GetOrders_DefaultsWhenPageAndLimitMissing(t *testing.T) {
	db := newTestDB(t)
	repo := order.NewOrderRepository(db)

	seedOrder(t, db, 1, "pending", "standard_mail", 10, time.Now())

	orders, total, page, err := repo.GetOrders(context.Background(), order.OrderFilters{}, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page != 1 {
		t.Fatalf("expected default page 1, got %d", page)
	}
	if total != 1 || len(orders) != 1 {
		t.Fatalf("expected 1 order, got total=%d len=%d", total, len(orders))
	}
}

func TestOrderRepository_GetOrders_FilterByStatus(t *testing.T) {
	db := newTestDB(t)
	repo := order.NewOrderRepository(db)

	seedOrder(t, db, 1, "pending", "standard_mail", 10, time.Now())
	wantID := seedOrder(t, db, 1, "delivered", "standard_mail", 20, time.Now())

	orders, total, _, err := repo.GetOrders(context.Background(), order.OrderFilters{Status: "delivered"}, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 matching order, got %d", total)
	}
	if len(orders) != 1 || orders[0].ID != wantID {
		t.Fatalf("expected only order %d, got %+v", wantID, orders)
	}
}

func TestOrderRepository_GetOrders_FilterBySearch(t *testing.T) {
	db := newTestDB(t)
	repo := order.NewOrderRepository(db)

	seedOrder(t, db, 1, "pending", "express_mail", 10, time.Now())
	seedOrder(t, db, 1, "pending", "local_drone", 20, time.Now())

	orders, total, _, err := repo.GetOrders(context.Background(), order.OrderFilters{Search: "drone"}, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 matching order, got %d", total)
	}
	if len(orders) != 1 || orders[0].Carrier != "local_drone" {
		t.Fatalf("expected only the drone order, got %+v", orders)
	}
}

func TestOrderRepository_GetOrders_ScansAllFields(t *testing.T) {
	db := newTestDB(t)
	repo := order.NewOrderRepository(db)

	seedOrder(t, db, 7, "pending", "standard_mail", 99.5, time.Now())

	orders, _, _, err := repo.GetOrders(context.Background(), order.OrderFilters{}, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(orders))
	}
	got := orders[0]
	if got.UserID != 7 || got.TotalAmount != 99.5 || got.Carrier != "standard_mail" || got.Status != "pending" {
		t.Fatalf("scanned order does not match seeded values: %+v", got)
	}
}

func TestOrderRepository_GetOrders_ScopedToRequestingUser(t *testing.T) {
	db := newTestDB(t)
	repo := order.NewOrderRepository(db)

	seedOrder(t, db, 1, "pending", "standard_mail", 10, time.Now())
	seedOrder(t, db, 2, "pending", "standard_mail", 20, time.Now())

	orders, total, _, err := repo.GetOrders(context.Background(), order.OrderFilters{}, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected only user 1's order to be counted, got total=%d", total)
	}
	if len(orders) != 1 || orders[0].UserID != 1 {
		t.Fatalf("expected only user 1's order, got %+v", orders)
	}
}
