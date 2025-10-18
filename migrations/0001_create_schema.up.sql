-- +goose Up
CREATE TABLE IF NOT EXISTS orders (
    order_uid         text PRIMARY KEY,
    track_number      text NOT NULL,
    entry             text NOT NULL,
    locale            text,
    internal_signature text,
    customer_id       text,
    delivery_service  text,
    shardkey          text,
    sm_id             int,
    date_created      timestamptz NOT NULL,
    oof_shard         text,
    raw_payload       jsonb NOT NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS deliveries (
    order_uid  text PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
    name       text NOT NULL,
    phone      text,
    zip        text,
    city       text NOT NULL,
    address    text NOT NULL,
    region     text,
    email      text
);

CREATE TABLE IF NOT EXISTS payments (
    order_uid     text PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
    transaction   text NOT NULL,
    request_id    text,
    currency      text NOT NULL,
    provider      text NOT NULL,
    amount        int NOT NULL,
    payment_dt    bigint NOT NULL,
    bank          text,
    delivery_cost int,
    goods_total   int,
    custom_fee    int
);

CREATE TABLE IF NOT EXISTS items (
    id           bigserial PRIMARY KEY,
    order_uid    text NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE,
    chrt_id      bigint NOT NULL,
    track_number text NOT NULL,
    price        int NOT NULL,
    rid          text NOT NULL,
    name         text NOT NULL,
    sale         int,
    size         text,
    total_price  int,
    nm_id        bigint,
    brand        text,
    status       int
);

CREATE INDEX IF NOT EXISTS idx_items_order_uid ON items(order_uid);
CREATE INDEX IF NOT EXISTS idx_orders_date_created ON orders(date_created);

-- +goose Down
DROP INDEX IF EXISTS idx_items_order_uid;
DROP INDEX IF EXISTS idx_orders_date_created;

DROP TABLE IF EXISTS items;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS deliveries;
DROP TABLE IF EXISTS orders;
