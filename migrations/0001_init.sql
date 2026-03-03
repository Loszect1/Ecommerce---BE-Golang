-- Core schema for eCommerce platform

BEGIN;

-- Users & auth

CREATE TABLE users (
    id              BIGSERIAL PRIMARY KEY,
    email           VARCHAR(255) NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    full_name       VARCHAR(255),
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX ux_users_email_lower
    ON users (LOWER(email));

CREATE TABLE user_providers (
    id               BIGSERIAL PRIMARY KEY,
    user_id          BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider         VARCHAR(32) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX ux_user_providers_provider_user
    ON user_providers (provider, provider_user_id);

CREATE TABLE roles (
    id          SMALLSERIAL PRIMARY KEY,
    name        VARCHAR(64) NOT NULL UNIQUE,
    description TEXT
);

CREATE TABLE user_roles (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id SMALLINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE refresh_tokens (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       VARCHAR(255) NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Catalog

CREATE TABLE categories (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    slug       VARCHAR(255) NOT NULL UNIQUE,
    parent_id  BIGINT REFERENCES categories(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE products (
    id               BIGSERIAL PRIMARY KEY,
    slug             VARCHAR(255) NOT NULL UNIQUE,
    name             VARCHAR(255) NOT NULL,
    description      TEXT,
    price_cents      INTEGER NOT NULL CHECK (price_cents >= 0),
    currency_code    CHAR(3) NOT NULL DEFAULT 'USD',
    main_image_url   TEXT,
    is_active        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ix_products_is_active_created_at
    ON products (is_active, created_at DESC);

CREATE TABLE product_images (
    id         BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    image_url  TEXT NOT NULL,
    alt_text   VARCHAR(255),
    sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE product_categories (
    product_id  BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    category_id BIGINT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (product_id, category_id)
);

-- Inventory

CREATE TABLE inventory (
    product_id      BIGINT PRIMARY KEY REFERENCES products(id) ON DELETE CASCADE,
    stock           INTEGER NOT NULL CHECK (stock >= 0),
    reserved_stock  INTEGER NOT NULL DEFAULT 0 CHECK (reserved_stock >= 0),
    version         INTEGER NOT NULL DEFAULT 1,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Carts

CREATE TABLE carts (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT REFERENCES users(id) ON DELETE SET NULL,
    session_id  VARCHAR(64),
    status      VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX ux_carts_session_active
    ON carts (session_id)
    WHERE status = 'active' AND session_id IS NOT NULL;

CREATE UNIQUE INDEX ux_carts_user_active
    ON carts (user_id)
    WHERE status = 'active' AND user_id IS NOT NULL;

CREATE TABLE cart_items (
    id                    BIGSERIAL PRIMARY KEY,
    cart_id               BIGINT NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    product_id            BIGINT NOT NULL REFERENCES products(id),
    quantity              INTEGER NOT NULL CHECK (quantity > 0),
    price_cents_snapshot  INTEGER NOT NULL CHECK (price_cents_snapshot >= 0),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX ux_cart_items_cart_product
    ON cart_items (cart_id, product_id);

-- Orders & payments

CREATE TABLE orders (
    id                    BIGSERIAL PRIMARY KEY,
    user_id               BIGINT REFERENCES users(id) ON DELETE SET NULL,
    status                VARCHAR(32) NOT NULL, -- pending, paid, cancelled, refunded...
    total_amount_cents    INTEGER NOT NULL CHECK (total_amount_cents >= 0),
    currency_code         CHAR(3) NOT NULL DEFAULT 'USD',
    payment_status        VARCHAR(32) NOT NULL DEFAULT 'pending',
    payment_provider      VARCHAR(32),
    payment_reference     VARCHAR(255),
    placed_at             TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ix_orders_user_created_at
    ON orders (user_id, created_at DESC);

CREATE TABLE order_items (
    id                 BIGSERIAL PRIMARY KEY,
    order_id           BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id         BIGINT NOT NULL REFERENCES products(id),
    quantity           INTEGER NOT NULL CHECK (quantity > 0),
    unit_price_cents   INTEGER NOT NULL CHECK (unit_price_cents >= 0),
    total_price_cents  INTEGER NOT NULL CHECK (total_price_cents >= 0)
);

CREATE TABLE payments (
    id                  BIGSERIAL PRIMARY KEY,
    order_id            BIGINT NOT NULL UNIQUE REFERENCES orders(id) ON DELETE CASCADE,
    provider            VARCHAR(32) NOT NULL, -- stripe, ...
    provider_payment_id VARCHAR(255),
    amount_cents        INTEGER NOT NULL CHECK (amount_cents >= 0),
    currency_code       CHAR(3) NOT NULL DEFAULT 'USD',
    status              VARCHAR(32) NOT NULL, -- created, succeeded, failed, refunded...
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE payment_logs (
    id          BIGSERIAL PRIMARY KEY,
    payment_id  BIGINT NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    event_type  VARCHAR(64) NOT NULL,
    raw_payload JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Addresses (user & order level)

CREATE TABLE addresses (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT REFERENCES users(id) ON DELETE SET NULL,
    order_id      BIGINT REFERENCES orders(id) ON DELETE CASCADE,
    address_type  VARCHAR(16) NOT NULL, -- billing / shipping
    full_name     VARCHAR(255),
    line1         VARCHAR(255) NOT NULL,
    line2         VARCHAR(255),
    city          VARCHAR(128) NOT NULL,
    region        VARCHAR(128),
    postal_code   VARCHAR(32),
    country_code  CHAR(2) NOT NULL,
    phone         VARCHAR(32),
    is_default    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMIT;

