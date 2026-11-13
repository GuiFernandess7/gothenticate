CREATE TABLE users (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    name     TEXT NOT NULL,
    email    TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    status   TEXT NOT NULL DEFAULT 'active',
    retries  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE products (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    unit_price REAL NOT NULL,
    stock      INTEGER NOT NULL
);

CREATE TABLE orders (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id      INTEGER NOT NULL REFERENCES users(id),
    status       TEXT NOT NULL DEFAULT 'pending',
    total_amount REAL NOT NULL,
    weight       REAL NOT NULL DEFAULT 0,
    zip          TEXT NOT NULL,
    carrier      TEXT NOT NULL,
    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE order_items (
    id         TEXT PRIMARY KEY,
    order_id   INTEGER NOT NULL REFERENCES orders(id),
    product_id TEXT NOT NULL REFERENCES products(id),
    quantity   INTEGER NOT NULL,
    unit_price REAL NOT NULL
);
