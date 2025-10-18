package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/kosovrzn/wb-tech-l0/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:generate moq -pkg mocks -out ../mocks/repository_mock.go . Repository

type Repository interface {
	UpsertOrder(ctx context.Context, o *domain.Order, rawJSON []byte) error
	GetOrderRaw(ctx context.Context, id string) ([]byte, error)
	Warmup(ctx context.Context, limit int) (map[string][]byte, error)
}

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

func (p *Postgres) UpsertOrder(ctx context.Context, o *domain.Order, rawJSON []byte) error {
	if o.OrderUID == "" {
		return errors.New("empty order_uid")
	}
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
INSERT INTO orders (
  order_uid, track_number, entry, locale, internal_signature, customer_id,
  delivery_service, shardkey, sm_id, date_created, oof_shard, raw_payload, updated_at
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12, now()
)
ON CONFLICT (order_uid) DO UPDATE SET
  track_number = EXCLUDED.track_number,
  entry        = EXCLUDED.entry,
  locale       = EXCLUDED.locale,
  internal_signature = EXCLUDED.internal_signature,
  customer_id  = EXCLUDED.customer_id,
  delivery_service = EXCLUDED.delivery_service,
  shardkey     = EXCLUDED.shardkey,
  sm_id        = EXCLUDED.sm_id,
  date_created = EXCLUDED.date_created,
  oof_shard    = EXCLUDED.oof_shard,
  raw_payload  = EXCLUDED.raw_payload,
  updated_at   = now();
`, o.OrderUID, o.TrackNumber, o.Entry, o.Locale, o.InternalSignature, o.CustomerID,
		o.DeliveryService, o.Shardkey, o.SmID, o.DateCreated, o.OofShard, json.RawMessage(rawJSON))
	if err != nil {
		return fmt.Errorf("orders upsert: %w", err)
	}

	_, err = tx.Exec(ctx, `
INSERT INTO deliveries (
  order_uid, name, phone, zip, city, address, region, email
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,$8
)
ON CONFLICT (order_uid) DO UPDATE SET
  name=$2, phone=$3, zip=$4, city=$5, address=$6, region=$7, email=$8;
`, o.OrderUID, o.Delivery.Name, o.Delivery.Phone, o.Delivery.Zip, o.Delivery.City,
		o.Delivery.Address, o.Delivery.Region, o.Delivery.Email)
	if err != nil {
		return fmt.Errorf("deliveries upsert: %w", err)
	}

	_, err = tx.Exec(ctx, `
INSERT INTO payments (
  order_uid, transaction, request_id, currency, provider, amount,
  payment_dt, bank, delivery_cost, goods_total, custom_fee
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11
)
ON CONFLICT (order_uid) DO UPDATE SET
  transaction=$2, request_id=$3, currency=$4, provider=$5, amount=$6,
  payment_dt=$7, bank=$8, delivery_cost=$9, goods_total=$10, custom_fee=$11;
`, o.OrderUID, o.Payment.Transaction, o.Payment.RequestID, o.Payment.Currency, o.Payment.Provider,
		o.Payment.Amount, o.Payment.PaymentDT, o.Payment.Bank, o.Payment.DeliveryCost, o.Payment.GoodsTotal, o.Payment.CustomFee)
	if err != nil {
		return fmt.Errorf("payments upsert: %w", err)
	}

	_, err = tx.Exec(ctx, `DELETE FROM items WHERE order_uid = $1`, o.OrderUID)
	if err != nil {
		return fmt.Errorf("items delete: %w", err)
	}
	for _, it := range o.Items {
		_, err = tx.Exec(ctx, `
INSERT INTO items (
  order_uid, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12
)`, o.OrderUID, it.ChrtID, it.TrackNumber, it.Price, it.RID, it.Name, it.Sale, it.Size, it.TotalPrice, it.NmID, it.Brand, it.Status)
		if err != nil {
			return fmt.Errorf("items insert: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (p *Postgres) GetOrderRaw(ctx context.Context, id string) ([]byte, error) {
	var raw []byte
	err := p.pool.QueryRow(ctx, `SELECT raw_payload FROM orders WHERE order_uid=$1`, id).Scan(&raw)
	return raw, err
}

func (p *Postgres) Warmup(ctx context.Context, limit int) (map[string][]byte, error) {
	q := `SELECT order_uid, raw_payload FROM orders ORDER BY updated_at DESC`
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := p.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string][]byte, 1024)
	var id string
	var raw []byte
	for rows.Next() {
		if err := rows.Scan(&id, &raw); err != nil {
			return nil, err
		}
		buf := make([]byte, len(raw))
		copy(buf, raw)
		out[id] = buf
	}
	return out, rows.Err()
}
