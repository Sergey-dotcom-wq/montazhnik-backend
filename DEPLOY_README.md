# Деплой стека в облако (Docker)

Поднимает полный стек одним `docker compose`:
PostgreSQL + SeaweedFS (S3) + Go-бэкенд + nginx (сайты и reverse-proxy API).

## Состав

| Файл | Назначение |
|---|---|
| `docker-compose.yml` | Сервисы: postgres, seaweedfs-s3, backend, nginx |
| `nginx/nginx.conf` | Раздача статики (dist/privacy) + прокси /api на backend |
| `.env.example` | Шаблон секретов (скопировать в `.env`) |

## Быстрый старт на VPS (Ubuntu)

```bash
# 1. Docker
curl -fsSL https://get.docker.com | sh
sudo systemctl enable --now docker

# 2. Клонировать проект (или загрузить код)
git clone <ваш-репозиторий> && cd deploy/docker

# 3. Секреты
cp .env.example .env   # впишите реальные JWT_SECRET, пароли, S3_ENC_KEY

# 4. mTLS-ключи (если backend использует mTLS)
mkdir pki  # положите ca_root.crt, server.crt, server.key по путям из nginx.conf
mkdir nginx/ssl

# 5. Запустить
docker compose up -d --build
```

## Проверка

- Сайты: `http://<IP>/` (витрина), `http://<IP>/privacy/` (политика)
- API: `curl http://localhost:8080/api/v1/time`
- S3: `http://localhost:9000` (SeaweedFS S3)

## Миграции БД

Backend применяет миграции при старте (см. `backend/migrations`). Если база пустая —
поднимается автоматически; для переноса данных используйте `pg_dump`/`psql`.

## Безопасность

- Не публикуйте `:5432` и `:9000` наружу (в compose они не проброшены наружу).
- SSH на VPS — только по ключу.
- Пароли/секреты — только в `.env` (вне git).