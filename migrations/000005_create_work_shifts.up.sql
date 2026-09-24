CREATE TABLE IF NOT EXISTS work_shifts (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(id),
    org_id           UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    status           TEXT NOT NULL DEFAULT 'none' CHECK (status IN ('none','active','paused','ended','locked')),
    started_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    paused_at        TIMESTAMPTZ,
    resumed_at       TIMESTAMPTZ,
    ended_at         TIMESTAMPTZ,
    last_active      TIMESTAMPTZ NOT NULL DEFAULT now(),
    sleep_until      TIMESTAMPTZ,
    unlock_requested BOOLEAN NOT NULL DEFAULT false,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_work_shifts_user_active ON work_shifts(user_id, status);
CREATE INDEX IF NOT EXISTS idx_work_shifts_org_status ON work_shifts(org_id, status);