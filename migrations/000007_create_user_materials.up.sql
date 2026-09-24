CREATE TABLE IF NOT EXISTS user_materials (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users(id),
    org_id         UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    item_id        UUID NOT NULL REFERENCES warehouse_items(id),
    name           TEXT NOT NULL,
    unit           TEXT NOT NULL DEFAULT '',
    quantity       DOUBLE PRECISION NOT NULL DEFAULT 0,
    assigned_to_job TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, item_id)
);

CREATE INDEX IF NOT EXISTS idx_user_materials_user ON user_materials(user_id);