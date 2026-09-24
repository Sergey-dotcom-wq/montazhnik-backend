CREATE TABLE IF NOT EXISTS critical_alerts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    message         TEXT NOT NULL DEFAULT '',
    severity        TEXT NOT NULL DEFAULT 'critical' CHECK (severity IN ('info','warning','critical')),
    status          TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','active','acknowledged','expired')),
    created_by      UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    acknowledged_by UUID REFERENCES users(id),
    acknowledged_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_critical_alerts_org_status ON critical_alerts(org_id, status);