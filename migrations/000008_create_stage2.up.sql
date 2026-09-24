-- Миграция 000008: синхронизация Stage 2 с сервером.
-- Граф сварок, тепловые и акустические замеры от инженеров → отчёт диспетчера.

-- Точки сварки (граф трассы)
CREATE TABLE IF NOT EXISTS splice_points (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id),
    route_id    TEXT NOT NULL DEFAULT '',
    sequence    INTEGER NOT NULL DEFAULT 0,
    sleeve_id   TEXT NOT NULL DEFAULT '',
    loss_db     DOUBLE PRECISION NOT NULL DEFAULT 0,
    lat         DOUBLE PRECISION,
    lng         DOUBLE PRECISION,
    note        TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_splice_points_org ON splice_points(org_id, route_id);

-- Тепловые замеры (ΔT контакта)
CREATE TABLE IF NOT EXISTS thermal_readings (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id         UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id        UUID NOT NULL REFERENCES users(id),
    contact_temp_c DOUBLE PRECISION NOT NULL,
    ambient_temp_c DOUBLE PRECISION NOT NULL,
    delta_c        DOUBLE PRECISION NOT NULL,
    location       TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_thermal_readings_org ON thermal_readings(org_id, created_at);

-- Акустические замеры (уровень шума)
CREATE TABLE IF NOT EXISTS acoustic_readings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id),
    level_db    DOUBLE PRECISION NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_acoustic_readings_org ON acoustic_readings(org_id, created_at);