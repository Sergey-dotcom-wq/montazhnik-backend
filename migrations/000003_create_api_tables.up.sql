-- Миграция 000003: Таблицы для топлива, склада, дорог
-- Эндпоинты, которые ожидает Android-клиент

-- Учёт топлива (POST /api/v1/fuel)
CREATE TABLE fuel_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    odometer_km DOUBLE PRECISION NOT NULL,
    liters DOUBLE PRECISION NOT NULL,
    price_per_liter DOUBLE PRECISION NOT NULL,
    timestamp_ms BIGINT NOT NULL,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Склад: номенклатура (GET /api/v1/warehouse/items)
CREATE TABLE warehouse_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    unit VARCHAR(20) NOT NULL,
    quantity DOUBLE PRECISION NOT NULL DEFAULT 0,
    min_threshold DOUBLE PRECISION NOT NULL DEFAULT 10,
    category VARCHAR(100) NOT NULL DEFAULT 'common',
    can_be_requested BOOLEAN NOT NULL DEFAULT TRUE,
    expected_delivery_days INT,
    code VARCHAR(100) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Заявки на склад (POST /api/v1/warehouse/requests)
CREATE TABLE warehouse_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id UUID REFERENCES warehouse_items(id),
    quantity DOUBLE PRECISION NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'delivered', 'cancelled')),
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Дороги: маршруты (GET /api/v1/road/route)
CREATE TABLE road_routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    extra_minutes INT NOT NULL DEFAULT 0,
    note TEXT,
    assigned_to UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Дорожные события (GET /api/v1/road/conditions)
CREATE TABLE road_conditions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    route_id UUID REFERENCES road_routes(id),
    type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    description TEXT,
    source VARCHAR(50) NOT NULL DEFAULT 'user',
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    reported_at_ms BIGINT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Индексы
CREATE INDEX idx_fuel_records_user ON fuel_records(created_by);
CREATE INDEX idx_fuel_records_time ON fuel_records(timestamp_ms);
CREATE INDEX idx_warehouse_items_category ON warehouse_items(category);
CREATE INDEX idx_warehouse_requests_status ON warehouse_requests(status);
CREATE INDEX idx_road_routes_assigned ON road_routes(assigned_to);
CREATE INDEX idx_road_conditions_route ON road_conditions(route_id);