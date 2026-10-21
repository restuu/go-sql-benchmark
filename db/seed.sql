TRUNCATE orders, customers, suppliers, models RESTART IDENTITY;

INSERT INTO models (name, title, fax, web, age, "right", counter)
SELECT 'Orm Benchmark', 'Just a Benchmark for fun', '99909990', 'http://blog.milkpod29.me', 100, TRUE, 1000
FROM generate_series(1, 10000);

INSERT INTO customers (name, email, city)
SELECT 'customer ' || i, 'customer' || i || '@example.com', 'city ' || (i % 50)
FROM generate_series(1, 1000) AS i;

INSERT INTO suppliers (name, email, city)
SELECT 'supplier ' || i, 'supplier' || i || '@example.com', 'city ' || (i % 50)
FROM generate_series(1, 1000) AS i;

INSERT INTO orders (customer_id, status, total, created_at)
SELECT (i % 1000) + 1,
       (ARRAY['new', 'paid', 'shipped', 'done'])[(i % 4) + 1],
       (i % 500) + 1,
       TIMESTAMPTZ '2026-01-01 00:00:00+00' + i * INTERVAL '1 minute'
FROM generate_series(1, 10000) AS i;

ANALYZE;
