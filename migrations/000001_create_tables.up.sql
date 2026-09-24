-- Миграция 000001: Создание основных таблиц
-- Сквозная история: матрица постоянных состояний (jobs) + матрица динамических изменений (job_events)

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- 1. Таблица нарядов (Матрица ПОСТОЯННЫХ состояний)
--    Неизменяемые атрибуты: адрес, плановое время, ответственный
-- ============================================================
CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id VARCHAR(100),                -- ID из 1С (для интеграции)
    address TEXT NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'new'
        CHECK (status IN ('new', 'assigned', 'in_progress', 'completed', 'cancelled')),
    assigned_to UUID,
    planned_start TIMESTAMPTZ,
    planned_end TIMESTAMPTZ,
    actual_start TIMESTAMPTZ,
    actual_end TIMESTAMPTZ,
    location VARCHAR(100),                    -- "lat,lon" последней фиксации
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    updated_by UUID
);

CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_assigned ON jobs(assigned_to);
CREATE INDEX idx_jobs_external ON jobs(external_id);
CREATE INDEX idx_jobs_planned ON jobs(planned_start, planned_end);

-- ============================================================
-- 2. Таблица событий (Матрица ДИНАМИЧЕСКИХ изменений)
--    Append-only журнал с SHA-256 цепочкой хешей.
--    Каждое событие связано с предыдущим через previous_hash.
--    Это позволяет восстановить полную историю через 100 лет.
-- ============================================================
CREATE TABLE job_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL
        CHECK (event_type IN ('photo_uploaded', 'measurement_taken', 'status_changed', 'material_scanned')),
    payload JSONB NOT NULL DEFAULT '{}',
    previous_hash VARCHAR(64) NOT NULL DEFAULT '0000000000000000000000000000000000000000000000000000000000000000',
    current_hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID
);

CREATE INDEX idx_job_events_job_id ON job_events(job_id);
CREATE INDEX idx_job_events_hash ON job_events(current_hash);
CREATE INDEX idx_job_events_created ON job_events(created_at);

-- Уникальность: для одного наряда цепочка не должна иметь два события с одинаковым previous_hash
CREATE UNIQUE INDEX idx_job_events_prev_hash ON job_events(job_id, previous_hash);

-- ============================================================
-- 3. Таблица материалов (Справочник)
-- ============================================================
CREATE TABLE materials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sku VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    unit VARCHAR(20) NOT NULL,                -- шт, м, бухта
    price DECIMAL(12, 2) NOT NULL,
    vat DECIMAL(5, 2) DEFAULT 20.00,
    category VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_materials_sku ON materials(sku);

-- ============================================================
-- 4. Таблица использованных материалов в наряде
--    Связывает статику (наряд) с динамикой (расход)
-- ============================================================
CREATE TABLE job_materials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    material_id UUID NOT NULL REFERENCES materials(id),
    quantity DECIMAL(10, 2) NOT NULL,
    action VARCHAR(20) NOT NULL CHECK (action IN ('consumed', 'returned')),
    scanned_at TIMESTAMPTZ DEFAULT NOW(),
    scanned_by UUID
);

CREATE INDEX idx_job_materials_job ON job_materials(job_id);

-- ============================================================
-- 5. Таблица партнеров (Партнерская программа)
-- ============================================================
CREATE TABLE partners (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    inn VARCHAR(12) UNIQUE NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    referral_code VARCHAR(10) UNIQUE NOT NULL,
    payout_percent DECIMAL(5, 2) NOT NULL DEFAULT 50.00,
    bank_account VARCHAR(20),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'blocked', 'closed')),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================
-- 6. Таблица рефералов (Кто кого привел)
-- ============================================================
CREATE TABLE referrals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    partner_id UUID NOT NULL REFERENCES partners(id),
    client_id UUID NOT NULL,
    subscription_id VARCHAR(100),
    amount DECIMAL(12, 2) NOT NULL,
    payout_status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (payout_status IN ('pending', 'paid', 'failed')),
    payout_date TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_referrals_partner ON referrals(partner_id);
CREATE INDEX idx_referrals_status ON referrals(payout_status);