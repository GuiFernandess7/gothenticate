package order

import (
	"context"
	"database/sql"
)


type OrderRepositoryInterface interface {
    GetOrders(ctx context.Context, filters OrderFilters) ([]Order, int, int, error)
	AddOrder(ctx context.Context, order Order)
	GetOrderByID(ctx context.Context, id int)
	UpdateOrderStatus(ctx context.Context, id int, status string)
	DeleteOrder(ctx context.Context, id int)
}


type OrderRepository struct {
	DB *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
    return &OrderRepository{DB: db}
}

func (r *OrderRepository) AddOrder(ctx context.Context, order Order) (*Order, error) {
	result, err := r.DB.ExecContext(
		ctx,
		`INSERT INTO orders (user_id, status, total_amount, weight, zip, carrier) VALUES (?, ?, ?, ?, ?, ?)`,
		order.UserID, order.Status, order.TotalAmount, order.Weight, order.Zip, order.Carrier,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return r.GetOrderByID(ctx, id)
}

func (r *OrderRepository) GetOrderByID(
    ctx context.Context,
    id int64,
) (*Order, error) {

    order := &Order{}

    err := r.DB.QueryRowContext(
        ctx,
        `SELECT id, user_id, status, total_amount, weight, zip, carrier, created_at
         FROM orders WHERE id = ?`,
        id,
    ).Scan(
        &order.ID,
        &order.UserID,
        &order.Status,
        &order.TotalAmount,
        &order.Weight,
        &order.Zip,
        &order.Carrier,
        &order.CreatedAt,
    )

    if err != nil {
        return nil, err
    }

    return order, nil
}


func (r *OrderRepository) GetOrders(ctx context.Context, filters OrderFilters, userID int64) ([]Order, int, int, error) {
	// 1. Monta a cláusula WHERE de forma incremental, sempre com placeholders (?)
	//    -- nunca interpolando o valor do filtro direto na string da query.
	where := "WHERE 1=1"
	args := []any{}

	if filters.Status != "" {
		where += " AND status = ?"
		args = append(args, filters.Status)
	}
	if filters.Search != "" {
		where += " AND carrier LIKE ?"
		args = append(args, "%"+filters.Search+"%")
	}
	where += " AND user_id = ?"
	args = append(args, userID)

	// 2. Conta o total de linhas que batem com o filtro (sem LIMIT/OFFSET),
	//    pro client saber quantas páginas existem.
	var total int
	countQuery := "SELECT COUNT(*) FROM orders " + where
	if err := r.DB.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, 0, err
	}

	// 3. Normaliza paginação (evita LIMIT 0 ou OFFSET negativo).
	limit := filters.Limit
	if limit <= 0 {
		limit = 10
	}
	page := filters.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	// 4. Busca a página pedida.
	query := "SELECT id, user_id, status, total_amount, weight, zip, carrier, created_at FROM orders " +
		where + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()

	orders := []Order{}
	for rows.Next() {
		var o Order
		if err := rows.Scan(
			&o.ID, &o.UserID, &o.Status, &o.TotalAmount,
			&o.Weight, &o.Zip, &o.Carrier, &o.CreatedAt,
		); err != nil {
			return nil, 0, 0, err
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, 0, err
	}

	return orders, total, page, nil
}

func (r *OrderRepository) UpdateOrderStatus(ctx context.Context, orderID int64, status string) (*Order, error){
	result, err := r.DB.ExecContext(
        ctx,
        `UPDATE orders SET status = ? WHERE id = ?`,
        status,
        orderID,
    )
	if err != nil {
        return nil, err
    }

	rowsAffected, err := result.RowsAffected()
	if err != nil {
        return nil, err
    }
	if rowsAffected == 0 {
        return nil, sql.ErrNoRows
    }

	return r.GetOrderByID(ctx, orderID)
}

func (r *OrderRepository) DeleteOrder(ctx context.Context, orderID int64)(error){
	result, err := r.DB.ExecContext(
		ctx,
        `DELETE FROM orders WHERE id = ?`,
        orderID,
	)

	if err != nil {
        return err
    }

	rowsAffected, err := result.RowsAffected()
	if err != nil {
        return err
    }

	if rowsAffected == 0 {
        return sql.ErrNoRows
    }
	return nil
}