

create table if not exists products(
  id BIGSERIAL primary key,
  name varchar(255) not null check(length(TRIM(name))>0),
  sku varchar(64) not null  check (sku ~ '^[A-Z0-9]+$'),
  description text,
  price numeric(10,2) not null check(price>=0.00),
  stock_quantity int not null default 0 check(stock_quantity>=0),
 status varchar(32) not null default 'draft' check(status in ('draft','active','archived')),
 created_at TIMESTAMPTZ not null default now(),
 updated_at TIMESTAMPTZ not null default now()


);

create unique INDEX idx_products_sku on products(sku);
CREATE INDEX idx_products_status
ON products(status);