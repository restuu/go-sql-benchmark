-- Single-table scenarios (INSERT/UPDATE/DELETE/SELECT). Same shape as upstream go-orm-benchmarks.
DROP TABLE IF EXISTS orders, customers, suppliers, models;

CREATE TABLE models (
    id      SERIAL  NOT NULL,
    name    TEXT    NOT NULL,
    title   TEXT    NOT NULL,
    fax     TEXT    NOT NULL,
    web     TEXT    NOT NULL,
    age     INTEGER NOT NULL,
    "right" BOOLEAN NOT NULL,
    counter BIGINT  NOT NULL,
    CONSTRAINT models_pkey PRIMARY KEY (id)
);

-- JOIN scenario: orders JOIN customers. UNION scenario: customers UNION suppliers (same column shape).
CREATE TABLE customers (
    id    SERIAL PRIMARY KEY,
    name  TEXT NOT NULL,
    email TEXT NOT NULL,
    city  TEXT NOT NULL
);
CREATE TABLE suppliers (
    id    SERIAL PRIMARY KEY,
    name  TEXT NOT NULL,
    email TEXT NOT NULL,
    city  TEXT NOT NULL
);
CREATE TABLE orders (
    id          SERIAL      PRIMARY KEY,
    customer_id INTEGER     NOT NULL REFERENCES customers (id),
    status      TEXT        NOT NULL,
    total       INTEGER     NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX orders_customer_id_idx ON orders (customer_id);
