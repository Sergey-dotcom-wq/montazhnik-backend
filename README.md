# Монтажник Связи — backend (сервер)

Серверная часть экосистемы «Монтажник Связи»: витрина, учёт, партнёрка,
раздача приложений (APK/exe), защищённый API и интеграция с S3-хранилищем.
Написана на Go.

Смежные репозитории: сайт/витрина, мобильные приложения, программы экосистемы
(см. README проекта).

## Состав стека

| Компонент | Назначение |
|---|---|
| **backend** (Go) | HTTP :8080, защищённый mTLS :8443, API `/api/v1/*` |
| **PostgreSQL** | основная БД (`montazhnik`) |
| **SeaweedFS (S3)** | объектное хранилище (`:9000` S3, `:9080` volume, `:9333` master) |
| **nginx** | раздача статики/приложений + reverse-proxy API + HTTPS |

## Демонстрация/продакшн

Бэкенд разворачивается через `docker compose` вместе с БД, S3 и nginx.
Полная инструкция по деплою на VPS — в [DEPLOY_README.md](DEPLOY_README.md).

## Структура

```
backend/
├── cmd/          # точка входа (server)
├── internal/     # хендлеры, сервисы, правила шлюза
├── migrations/   # миграции БД (применяются при старте)
├── nginx/        # конфиг nginx (сайты + reverse-proxy)
├── pki/          # ключи mTLS (НЕ в git)
├── .env.example  # шаблон секретов (копировать в .env)
├── docker-compose.yml, Dockerfile, deploy.sh
└── DEPLOY_README.md
```

## Быстрый старт (локально)

```bash
# 1. Секреты
cp .env.example .env   # впишите реальные JWT_SECRET, пароли, S3_ENC_KEY

# 2. Запуск всего стека
docker compose up -d --build
```

## Проверка

- API: `curl http://localhost:8080/api/v1/time`
- S3: `http://localhost:9000` (SeaweedFS S3, доступ по ключам из `.env`)
- Сайт (если настроен): `https://<домен>/`

## Переменные окружения (основные)

Все секреты задаются только через `.env` (вне git). Основные:

- `POSTGRES_PASSWORD`, `JWT_SECRET` — безопасность БД и токенов;
- `S3_BUCKET`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_RETENTION` — S3;
- `S3_ENC_KEY` — ключ шифрования S3 (Hex 32 байт);
- `PAYMENT_PROVIDER` (mock | sber | yookassa) и ключи платёжного провайдера;
- `GIGACHAT_*` — данные для LLM (GigaChat).

Полный список — в `.env.example`.

## Безопасность

- Секреты — только в `.env`, файл не попадает в git (см. `.gitignore`).
- Порт `:5432` и `:9000` не пробрасываются наружу в `docker-compose.yml`.
- HTTPS обязателен (витрина и раздача), авторизация — токен в `Authorization`.
- Ключи и сертификаты (`.pem/.key/.p12`, `pki/`) исключены из git.

## Лицензия

- Исходный код: свободная лицензия MIT — см. [LICENSE](LICENSE).
- Коммерческое использование и распространение — см.
  [LICENSE-COMMERCIAL](LICENSE-COMMERCIAL.txt).
- Сторонние компоненты — см. [THIRD_PARTY_NOTICES](THIRD_PARTY_NOTICES.txt).