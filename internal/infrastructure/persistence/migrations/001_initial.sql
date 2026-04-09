PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;
PRAGMA foreign_keys=ON;

CREATE TABLE IF NOT EXISTS products (
    sku              TEXT PRIMARY KEY,
    name             TEXT NOT NULL,
    barcode          TEXT UNIQUE,
    price_cents      INTEGER NOT NULL CHECK(price_cents >= 0),
    cost_cents       INTEGER NOT NULL DEFAULT 0,
    cost_total_cents INTEGER NOT NULL DEFAULT 0,
    stock_actual     INTEGER NOT NULL DEFAULT 0,
    stock_minimo     INTEGER NOT NULL DEFAULT 0,
    unit_type        TEXT NOT NULL DEFAULT 'PIEZA',
    unit_factor      INTEGER NOT NULL DEFAULT 1,
    active           INTEGER NOT NULL DEFAULT 1,
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS users (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    pin_hash    TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT 'CAJERO',
    active      INTEGER NOT NULL DEFAULT 1,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS cash_sessions (
    id              TEXT PRIMARY KEY,
    cajero_id       TEXT NOT NULL REFERENCES users(id),
    terminal_id     TEXT NOT NULL,
    initial_cash    INTEGER NOT NULL DEFAULT 0,
    total_sales     INTEGER NOT NULL DEFAULT 0,
    withdrawals     INTEGER NOT NULL DEFAULT 0,
    expected_cash   INTEGER NOT NULL DEFAULT 0,
    real_cash       INTEGER NOT NULL DEFAULT 0,
    difference      INTEGER NOT NULL DEFAULT 0,
    opened_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
    closed_at       DATETIME,
    signature_hash  TEXT
);

CREATE TABLE IF NOT EXISTS cash_movements (
    id          TEXT PRIMARY KEY,
    session_id  TEXT NOT NULL REFERENCES cash_sessions(id),
    amount      INTEGER NOT NULL,
    type        TEXT NOT NULL,
    reason      TEXT NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS local_receipts (
    id              TEXT PRIMARY KEY,
    session_id      TEXT NOT NULL REFERENCES cash_sessions(id),
    cajero_id       TEXT NOT NULL,
    terminal_id     TEXT NOT NULL,
    subtotal_cents  INTEGER NOT NULL,
    iva_cents       INTEGER NOT NULL,
    total_cents     INTEGER NOT NULL,
    payment_method  TEXT NOT NULL,
    paid_cents      INTEGER NOT NULL DEFAULT 0,
    change_cents    INTEGER NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'COMPLETED',
    fiscal_status   TEXT NOT NULL DEFAULT 'NO_FACTURA',
    fiscal_uuid     TEXT,
    signature_hash  TEXT NOT NULL,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_receipts_session ON local_receipts(session_id);
CREATE INDEX IF NOT EXISTS idx_receipts_created ON local_receipts(created_at);
CREATE INDEX IF NOT EXISTS idx_receipts_fiscal ON local_receipts(fiscal_status);

CREATE TABLE IF NOT EXISTS receipt_items (
    id              TEXT PRIMARY KEY,
    receipt_id      TEXT NOT NULL REFERENCES local_receipts(id),
    sku             TEXT NOT NULL,
    name            TEXT NOT NULL,
    quantity        INTEGER NOT NULL,
    price_cents     INTEGER NOT NULL,
    subtotal_cents  INTEGER NOT NULL,
    iva_cents       INTEGER NOT NULL,
    total_cents     INTEGER NOT NULL,
    cost_cents      INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS inventory_kardex (
    id              TEXT PRIMARY KEY,
    sku             TEXT NOT NULL,
    movement_type   TEXT NOT NULL,
    quantity        INTEGER NOT NULL,
    cost_cents      INTEGER NOT NULL,
    balance_after   INTEGER NOT NULL,
    reference_id    TEXT,
    notes           TEXT,
    cajero_id       TEXT,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_kardex_sku ON inventory_kardex(sku, created_at);
CREATE INDEX IF NOT EXISTS idx_kardex_ref ON inventory_kardex(reference_id);

CREATE TABLE IF NOT EXISTS suspended_sales (
    id              TEXT PRIMARY KEY,
    cajero_id       TEXT NOT NULL,
    session_id      TEXT NOT NULL,
    items_json      TEXT NOT NULL,
    total_cents     INTEGER NOT NULL,
    suspended_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at      DATETIME NOT NULL,
    status          TEXT NOT NULL DEFAULT 'active'
);

CREATE TABLE IF NOT EXISTS stock_reservations (
    id          TEXT PRIMARY KEY,
    sale_id     TEXT NOT NULL REFERENCES suspended_sales(id),
    sku         TEXT NOT NULL,
    quantity    INTEGER NOT NULL,
    expires_at  DATETIME NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_reservations_sku ON stock_reservations(sku, expires_at);
CREATE INDEX IF NOT EXISTS idx_suspended_expires ON suspended_sales(expires_at, status);

-- ─── OUTBOX DE SINCRONIZACIÓN ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS sync_outbox (
    id          TEXT PRIMARY KEY,
    entity_type TEXT NOT NULL,
    entity_id   TEXT NOT NULL,
    payload     TEXT NOT NULL,
    priority    INTEGER NOT NULL DEFAULT 2,
    status      TEXT NOT NULL DEFAULT 'pending',
    attempts    INTEGER NOT NULL DEFAULT 0,
    cloud_id    TEXT,
    last_error  TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    synced_at   DATETIME
);
CREATE INDEX IF NOT EXISTS idx_outbox_status ON sync_outbox(status, priority, created_at);

-- ─── LOGS DE AUDITORÍA ────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS audit_logs (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL,
    action          TEXT NOT NULL,
    description     TEXT,
    metadata        TEXT,
    signature_hash  TEXT,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ─── CFDI TIMBRADOS (FISCAL) ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS fiscal_timbres (
    id              TEXT PRIMARY KEY,
    receipt_id      TEXT NOT NULL UNIQUE REFERENCES local_receipts(id),
    cfdi_version    TEXT NOT NULL DEFAULT '4.0',
    serie_folio     TEXT NOT NULL,
    cfdi_xml        TEXT NOT NULL,
    signature_b64   TEXT NOT NULL,
    sat_uuid        TEXT,
    timbrado_at     DATETIME,
    status          TEXT NOT NULL DEFAULT 'pending',
    error_msg       TEXT,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_fiscal_status ON fiscal_timbres(status, created_at);
CREATE INDEX IF NOT EXISTS idx_fiscal_receipt ON fiscal_timbres(receipt_id);
CREATE INDEX IF NOT EXISTS idx_fiscal_sat_uuid ON fiscal_timbres(sat_uuid);
