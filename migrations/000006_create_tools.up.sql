CREATE TABLE IF NOT EXISTS tools (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id),
    name       TEXT NOT NULL,
    mac        TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'bound' CHECK (status IN ('bound','lost','removed')),
    battery    INTEGER NOT NULL DEFAULT -1 CHECK (battery BETWEEN -1 AND 100),
    last_seen  TIMESTAMPTZ NOT NULL DEFAULT now(),
    lost_lat   DOUBLE PRECISION,
    lost_lng   DOUBLE PRECISION,
    lost_at    TIMESTAMPTZ,
    bound_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (org_id, mac)
);

CREATE INDEX IF NOT EXISTS idx_tools_user ON tools(user_id, status);
CREATE INDEX IF NOT EXISTS idx_tools_org_mac ON tools(org_id, mac);