CREATE TABLE IF NOT EXISTS products (
    id          SERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    sku         TEXT NOT NULL UNIQUE,
    price_cents INTEGER NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO products (name, sku, price_cents) VALUES
    ('Field Notebook',  'FN-001', 1200),
    ('Ford Crossing Mug','FC-002', 1800),
    ('Ferryman Tee',    'FT-003', 2500)
ON CONFLICT (sku) DO NOTHING;
