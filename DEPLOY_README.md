# Деплой стека в облако (Docker)

Поднимает полный стек одним `docker compose`:
PostgreSQL + SeaweedFS (S3) + Go-бэкенд + nginx (сайты и reverse-proxy API + раздача приложений).

## Состав

| Файл | Назначение |
|---|---|
| `docker-compose.yml` | Сервисы: postgres, seaweedfs-s3, backend, nginx |
| `nginx/nginx.conf` | Раздача статики (dist/privacy) + прокси /api + раздача /downloads (APK/exe) + HTTPS |
| `.env.example` | Шаблон секретов (скопировать в `.env`) |

## Шаг 0. Регистрация и оплата VPS (РФ-хостинг)

1. Выберите провайдера: **reg.ru / beget / timeweb / vdsina** (оплата картой РФ, поддержка на русском).
2. Тариф: **2 vCPU / 2 ГБ RAM / 40 ГБ SSD / Ubuntu 22.04** — достаточно под портал + БД + раздачу APK.
3. Публичный IP + доступ по SSH (ключ). Обязательно: SSH только по ключу, выключить root-пароль.
4. Купите домен (рекомендуется у того же хостера), привяжите к IP сервера (A-запись).

> Настоящая LLM-модель на сервер НЕ ставится — она работает на устройстве (встроена в APK).
> Сервер нужен под витрину, учёт, партнёрку и раздачу установщиков (~37 МБ файлы, лёгкая нагрузка).

## Быстрый старт на VPS (Ubuntu)

```bash
# 1. Docker
curl -fsSL https://get.docker.com | sh
sudo systemctl enable --now docker

# 2. Клонировать проект (или загрузить код)
git clone <ваш-репозиторий> && cd deploy/docker

# 3. Секреты
cp .env.example .env   # впишите реальные JWT_SECRET, пароли, S3_ENC_KEY

# 4. HTTPS-сертификат (Let's Encrypt)
sudo apt install -y certbot
sudo certbot certonly --webroot -w /var/www/letsencrypt -d <ВАШ-ДОМЕН>
sudo ln -sf /etc/letsencrypt/live/<ВАШ-ДОМЕН>/fullchain.pem /etc/nginx/ssl/fullchain.pem
sudo ln -sf /etc/letsencrypt/live/<ВАШ-ДОМЕН>/privkey.pem  /etc/nginx/ssl/privkey.pem

# 5. Раздача приложений: положить установщики
mkdir -p /var/www/downloads
cp montazhnik-app.apk /var/www/downloads/          # APK «Монтажник Сети»
cp autodiagnost.apk   /var/www/downloads/           # APK «АвтоДиагност»
cp montazhnik.exe     /var/www/downloads/           # десктоп-версия базы (Windows)

# 6. mTLS-ключи (если backend использует mTLS)
mkdir pki
mkdir nginx/ssl

# 7. Запустить
docker compose up -d --build
```

## Проверка

- Сайт: `https://<домен>/` (витрина), `https://<домен>/privacy/` (политика)
- Раздача: `https://<домен>/downloads/montazhnik-app.apk`
- API: `curl https://<домен>/api/v1/time`
- S3: `http://localhost:9000` (SeaweedFS S3)

## База скачиваний (закрытый доступ)

Распространение установщиков защищено:
- `/downloads/` отдаёт файлы по прямому URL (можно закрыть за авторизацией).
- Список доступных приложений и их статус — через `/api/apps` (готовить секцию «База скачиваний»).
- Оплата/партнёрка — подключается на уровне backend (новые эндпоинты).

## Миграции БД

Backend применяет миграции при старте (см. `backend/migrations`). Если база пустая —
поднимается автоматически; для переноса данных используйте `pg_dump`/`psql`.

## Безопасность

- Не публикуйте `:5432` и `:9000` наружу (в compose они не проброшены наружу).
- SSH на VPS — только по ключу.
- Пароли/секреты — только в `.env` (вне git).
- HTTPS — обязателен (витрина и раздача), токен в заголовке `Authorization`.