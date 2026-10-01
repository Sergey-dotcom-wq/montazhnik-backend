#!/usr/bin/env bash
# =============================================================================
# Деплой стека «Монтажник Связи» на свежую Yandex Cloud ВМ (Ubuntu 24)
#
# Использование:
#   sudo bash deploy.sh                # обычный путь (установит Docker + клонирует)
#   sudo bash deploy.sh --no-git       # пропустить клонирование (если код уже здесь)
#
# После: откройте http://<IP>/ и http://<IP>/api/v1/time
# =============================================================================
set -euo pipefail

GIT_URL="${GIT_URL:-https://github.com/Sergey-dotcom-wq/Montazhnik-Svyazi-2.git}"
BASE_DIR="${BASE_DIR:-/opt}"                 # родительская папка
REPO_DIR="${REPO_DIR:-${BASE_DIR}/Montazhnik-Svyazi-2}"  # корень клона
COMPOSE_DIR="${COMPOSE_DIR:-${REPO_DIR}/backend}"        # где лежит docker-compose.yml
BRANCH="${BRANCH:-main}"

echo "==> Монтажник Связи: деплой на $(hostname) $(date -Iseconds)"

# ---- 1. Docker + Compose (если не установлены) ----
if ! command -v docker >/dev/null 2>&1; then
  echo "==> Устанавливаю Docker..."
  curl -fsSL https://get.docker.com | sh
  systemctl enable --now docker
fi
if ! docker compose version >/dev/null 2>&1; then
  echo "ERROR: docker compose недоступен (нужен плагин compose v2)."
  exit 1
fi

# ---- 2. Код (клонировать или использовать локальный) ----
if [ "${1:-}" = "--no-git" ] || [ -f "${COMPOSE_DIR}/docker-compose.yml" ]; then
  echo "==> Код уже присутствует (${COMPOSE_DIR}), пропускаю клонирование."
else
  echo "==> Клонирую ${GIT_URL} -> ${REPO_DIR}"
  mkdir -p "$(dirname "${REPO_DIR}")"
  git clone --branch "${BRANCH}" --depth 1 "${GIT_URL}" "${REPO_DIR}" 2>/dev/null \
    || git clone --depth 1 "${GIT_URL}" "${REPO_DIR}"
fi

cd "${COMPOSE_DIR}"

# ---- 2б. Сайты-статика (для nginx mounts ../../dist-site и ../../privacy-site) ----
# После клона compose смотрит на /opt/dist-site и /opt/privacy-site.
if [ -d "${REPO_DIR}/docs/site" ] && [ ! -d "${BASE_DIR}/dist-site" ]; then
  echo "==> Размещаю сайт-каталог как ${BASE_DIR}/dist-site"
  mkdir -p "${BASE_DIR}/dist-site"
  cp -r "${REPO_DIR}/docs/site/." "${BASE_DIR}/dist-site/"
fi
if [ ! -d "${BASE_DIR}/privacy-site" ]; then
  echo "==> Копирую политику как ${BASE_DIR}/privacy-site"
  mkdir -p "${BASE_DIR}/privacy-site"
  # политика загружается отдельно пользователем (или из docs/site при желании)
fi

# ---- 3. .env ----
if [ ! -f .env ]; then
  echo "==> Создаю .env из шаблона (впишите реальные секреты!)."
  cp .env.example .env
  # Генерируем случайные значения для обязательных секретов.
  if grep -q 'JWT_SECRET=' .env; then
    sed -i "s/^JWT_SECRET=.*/JWT_SECRET=$(head -c32 /dev/urandom | base64 | tr -d '=/+' | cut -c1-32)/" .env
  fi
  echo "!!! Отредактируйте ${COMPOSE_DIR}/.env : впишите GIGACHAT_CLIENT_ID/SECRET и S3_* ключи."
fi

# ---- 4. mTLS-ключи (опционально) ----
if [ ! -d pki ]; then
  echo "==> Папка pki отсутствует. Если API требует mTLS — положите сюда ca_root.crt, server.crt, server.key."
  mkdir -p pki
fi

# ---- 5. nginx SSL (опционально) ----
mkdir -p nginx/ssl

# ---- 6. Поднять стек ----
echo "==> docker compose build + up"
docker compose up -d --build

# ---- 7. Проверка ----
sleep 3
echo ""
echo "=============================================================================="
echo "  Стек запущен."
MRK_HTTPS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/time || true)
echo "  API /time  -> HTTP ${MRK_HTTPS:-текстовый ответ}"
echo "  Сайты: http://<IP>/ и http://<IP>/privacy/"
echo "  Логи: docker compose -f ${COMPOSE_DIR}/docker-compose.yml logs -f"
echo "=============================================================================="
exit 0