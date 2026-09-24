-- Миграция 000009: реестр фото-доказательств (anti-fraud).
-- Хеш каждого фото + координаты + доверенное время регистрируются
-- при съёмке; позже подставить старый кадр нельзя — хеш не совпадёт.

CREATE TABLE IF NOT EXISTS photo_evidence (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES users(id),
    order_id      TEXT NOT NULL DEFAULT '',
    photo_sha256  TEXT NOT NULL,          -- SHA-256 фото
    latitude      DOUBLE PRECISION,
    longitude     DOUBLE PRECISION,
    accuracy_m    DOUBLE PRECISION,
    device_time_ms BIGINT NOT NULL,       -- время устройства в момент съёмки
    server_time_ms BIGINT NOT NULL,       -- доверенное время сервера
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_photo_evidence_hash ON photo_evidence(photo_sha256);
CREATE INDEX IF NOT EXISTS idx_photo_evidence_org ON photo_evidence(org_id, created_at);