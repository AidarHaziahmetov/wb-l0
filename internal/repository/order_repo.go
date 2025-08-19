package repository

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"wb-l0-go/internal/domain"
)

type OrderRepository interface {
	Save(ctx context.Context, msg domain.Order) error
	ListUIDs(ctx context.Context, limit, offset int) ([]string, error)
	List(ctx context.Context, limit, offset int) ([]domain.Order, error)
	Get(ctx context.Context, orderUID string) (domain.Order, error)
	SaveWithTx(ctx context.Context, tx pgx.Tx, msg domain.Order) error
}

type PostgresOrderRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresOrderRepository(pool *pgxpool.Pool) *PostgresOrderRepository {
	return &PostgresOrderRepository{pool: pool}
}

func (r *PostgresOrderRepository) Save(ctx context.Context, msg domain.Order) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	const q = `INSERT INTO orders (order_uid, payload) VALUES ($1, $2::jsonb)
               ON CONFLICT (order_uid) DO UPDATE SET payload = EXCLUDED.payload`
	_, err = r.pool.Exec(ctx, q, msg.OrderUID, string(payload))
	return err
}

func (r *PostgresOrderRepository) SaveWithTx(ctx context.Context, tx pgx.Tx, msg domain.Order) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	const q = `INSERT INTO orders (order_uid, payload) VALUES ($1, $2::jsonb)
               ON CONFLICT (order_uid) DO UPDATE SET payload = EXCLUDED.payload`
	_, err = tx.Exec(ctx, q, msg.OrderUID, string(payload))
	return err
}

func (r *PostgresOrderRepository) ListUIDs(ctx context.Context, limit, offset int) ([]string, error) {
	const q = `SELECT order_uid FROM orders ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.pool.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]string, 0, limit)
	for rows.Next() {
		var orderUID string
		if err := rows.Scan(&orderUID); err != nil {
			return nil, err
		}
		orders = append(orders, orderUID)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return orders, nil
}

func (r *PostgresOrderRepository) List(ctx context.Context, limit, offset int) ([]domain.Order, error) {
	const q = `SELECT payload FROM orders ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.pool.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]domain.Order, 0, limit)
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var ord domain.Order
		if err := json.Unmarshal(raw, &ord); err != nil {
			return nil, err
		}
		orders = append(orders, ord)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return orders, nil
}

func (r *PostgresOrderRepository) Get(ctx context.Context, orderUID string) (domain.Order, error) {
	const q = `SELECT payload FROM orders WHERE order_uid = $1`
	var raw []byte
	err := r.pool.QueryRow(ctx, q, orderUID).Scan(&raw)
	if err != nil {
		return domain.Order{}, err
	}
	var ord domain.Order
	if err := json.Unmarshal(raw, &ord); err != nil {
		return domain.Order{}, err
	}
	return ord, nil
}
