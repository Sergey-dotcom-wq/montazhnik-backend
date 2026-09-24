package web

// DashboardHTML — сайт руководителя коллектива в стиле десктоп-клиента.
// Тёмная тема (#0f172a + голубой #38bdf8), карточки статистики, навигация.
// Работает на нашем бэкенде: вход по username, данные своего коллектива (org_id),
// авто-обновление каждые 8 секунд.
const DashboardHTML = `<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Монтажник Связи — Дашборд</title>
<style>
:root {
  --bg: #0f172a; --panel: #1e293b; --panel2: #273449; --border: #334155;
  --text: #e2e8f0; --muted: #94a3b8; --accent: #38bdf8; --accent2: #0ea5e9;
  --green: #22c55e; --red: #ef4444; --amber: #f59e0b;
}
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: 'Segoe UI', system-ui, sans-serif; background: var(--bg); color: var(--text); min-height: 100vh; }
#app { display: flex; flex-direction: column; min-height: 100vh; }
header { display: flex; align-items: center; gap: 8px; padding: 10px 16px; background: var(--panel); border-bottom: 1px solid var(--border); }
header h1 { font-size: 18px; margin-right: auto; }
header .sub { color: var(--muted); font-size: 12px; }
nav { display: flex; gap: 6px; flex-wrap: wrap; }
nav button { background: transparent; color: var(--muted); border: 1px solid transparent; padding: 8px 14px; border-radius: 8px; cursor: pointer; font-size: 14px; }
nav button:hover { color: var(--text); background: var(--panel2); }
nav button.active { background: var(--accent2); color: #fff; }
main { flex: 1; padding: 20px; overflow: auto; }
.panel { background: var(--panel); border: 1px solid var(--border); border-radius: 12px; padding: 16px; margin-bottom: 16px; }
.panel h2 { font-size: 15px; margin-bottom: 12px; color: var(--accent); }
.stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px,1fr)); gap: 12px; margin-bottom: 16px; }
.stat { background: var(--panel); border: 1px solid var(--border); border-radius: 12px; padding: 14px; }
.stat .num { font-size: 22px; font-weight: 700; }
.stat .lbl { color: var(--muted); font-size: 12px; margin-top: 4px; }
.stat .num.green { color: var(--green); }
.stat .num.red { color: var(--red); }
.stat .num.amber { color: var(--amber); }
.stat .num.blue { color: var(--accent); }
table { width: 100%; border-collapse: collapse; font-size: 13px; }
th, td { text-align: left; padding: 8px 10px; border-bottom: 1px solid var(--border); overflow-wrap: break-word; word-break: break-word; max-width: 200px; }
th { color: var(--muted); font-weight: 600; font-size: 12px; text-transform: uppercase; letter-spacing: .03em; }
tr:hover td { background: var(--panel2); }
.badge { padding: 3px 8px; border-radius: 20px; font-size: 11px; font-weight: 600; }
.badge.new { background: rgba(245,158,11,.2); color: var(--amber); }
.badge.in_progress { background: rgba(56,189,248,.2); color: var(--accent); }
.badge.completed { background: rgba(34,197,94,.2); color: var(--green); }
.badge.cancelled { background: rgba(239,68,68,.2); color: var(--red); }
.badge.pending { background: rgba(245,158,11,.2); color: var(--amber); }
.badge.paid { background: rgba(34,197,94,.2); color: var(--green); }
@media (max-width: 640px) {
  main { padding: 8px; }
  .panel { padding: 10px; }
  nav { gap: 4px; }
  nav button { padding: 8px 10px; font-size: 13px; }
  th, td { padding: 6px 6px; max-width: 140px; }
  input, select { font-size: 14px; }
}
#loginForm { max-width: 360px; margin: 10vh auto; }
#loginForm .panel { padding: 24px; }
#loginForm h2 { color: var(--accent); margin-bottom: 16px; }
input, select { background: var(--panel2); color: var(--text); border: 1px solid var(--border); border-radius: 8px; padding: 8px 10px; font-size: 14px; width: 100%; margin-bottom: 10px; outline: none; }
input:focus { border-color: var(--accent); }
button.btn { background: var(--accent2); color: #fff; border: none; padding: 10px 14px; border-radius: 8px; cursor: pointer; font-size: 14px; width: 100%; }
button.btn:hover { filter: brightness(1.1); }
#loginError { color: var(--red); font-size: 13px; margin-bottom: 10px; display: none; }
#userInfo { display: flex; align-items: center; gap: 8px; color: var(--muted); font-size: 12px; }
#userInfo button { background: var(--panel2); color: var(--text); border: 1px solid var(--border); border-radius: 8px; padding: 5px 10px; cursor: pointer; }
.loading { color: var(--muted); text-align: center; padding: 20px; }
@media (max-width: 600px) { th, td { padding: 6px 8px; } }
</style>
</head>
<body>
<div id="app">

  <div id="loginForm">
    <div class="panel">
      <h2>Вход в систему</h2>
      <input type="text" id="loginUsername" placeholder="Логин (лат., до 10)" autocomplete="username">
      <input type="password" id="loginPassword" placeholder="Пароль" autocomplete="current-password">
      <div id="loginError">Неверный логин или пароль</div>
      <button class="btn" onclick="login()">Войти</button>
    </div>
  </div>

  <div id="mainUI" style="display:none">
    <header>
      <h1>🔧 Монтажник Связи</h1>
      <span class="sub" id="userInfo"></span>
      <nav>
        <button onclick="loadPage('stats')" data-page="stats" class="active">📊 Статистика</button>
        <button onclick="loadPage('jobs')" data-page="jobs">📋 Наряды</button>
        <button onclick="loadPage('planner')" data-page="planner">🗓️ Планировщик</button>
        <button onclick="loadPage('warehouse')" data-page="warehouse">📦 Склад</button>
        <button onclick="loadPage('docs')" data-page="docs">📄 Акты КС</button>
        <button onclick="loadPage('audit')" data-page="audit">🕵️ Журнал</button>
        <button onclick="loadPage('alerts')" data-page="alerts">🚨 Оповестить</button>
        <button onclick="loadPage('referrals')" data-page="referrals">📈 Партнёры</button>
        <button onclick="loadPage('payments')" data-page="payments">💳 Платежи</button>
        <button onclick="loadPage('integrity')" data-page="integrity">🔗 Целостность</button>
        <button onclick="openPresentation()">📊 Презентация</button>
        <button onclick="logout()">Выйти</button>
      </nav>
    </header>
    <main id="content">
      <div class="loading">Загрузка...</div>
    </main>
  </div>

</div>

<script>
const API = '/api/v1';

function token() { return localStorage.getItem('dashboard_token'); }
function username() { return localStorage.getItem('dashboard_username') || ''; }

async function api(method, path, body) {
  const opts = { method, headers: { 'Content-Type': 'application/json' } };
  const t = token();
  if (t) opts.headers['Authorization'] = 'Bearer ' + t;
  if (body) opts.body = JSON.stringify(body);
  const r = await fetch(API + path, opts);
  if (r.status === 401) { logout(); throw new Error('auth'); }
  if (!r.ok) { const e = await r.text(); throw new Error(r.status + ': ' + e); }
  const ct = r.headers.get('content-type') || '';
  return ct.includes('json') ? r.json() : r.text();
}

async function login() {
  const u = document.getElementById('loginUsername').value.trim();
  const p = document.getElementById('loginPassword').value;
  document.getElementById('loginError').style.display = 'none';
  try {
    const r = await fetch(API + '/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: u, password: p })
    });
    if (!r.ok) { document.getElementById('loginError').style.display = 'block'; return; }
    const data = await r.json();
    localStorage.setItem('dashboard_token', data.token);
    localStorage.setItem('dashboard_username', data.user?.username || u);
    document.getElementById('loginForm').style.display = 'none';
    document.getElementById('mainUI').style.display = 'flex';
    document.getElementById('userInfo').textContent = data.user?.full_name || data.user?.username || u;
    loadPage('stats');
  } catch (e) { document.getElementById('loginError').style.display = 'block'; }
}

function logout() {
  localStorage.removeItem('dashboard_token');
  localStorage.removeItem('dashboard_username');
  location.reload();
}

let currentPage = 'stats';
async function loadPage(page) {
  currentPage = page;
  document.querySelectorAll('nav button').forEach(b => b.classList.toggle('active', b.dataset.page === page));
  document.getElementById('content').innerHTML = '<div class="loading">Загрузка...</div>';
  try {
    if (page === 'stats') await loadStats();
    else if (page === 'jobs') await loadJobs();
    else if (page === 'planner') await loadPlanner();
    else if (page === 'warehouse') await loadWarehouse();
    else if (page === 'docs') await loadDocs();
    else if (page === 'audit') await loadAudit();
    else if (page === 'alerts') await loadAlerts();
    else if (page === 'referrals') await loadReferrals();
    else if (page === 'payments') await loadPayments();
    else if (page === 'integrity') await loadIntegrity();
  } catch (e) {
    document.getElementById('content').innerHTML = '<div class="loading" style="color:var(--red)">Ошибка: ' + e.message + '</div>';
  }
}

async function loadStats() {
  let jobsCount = 0, whCount = 0, referralsPending = 0, paymentsSum = 0;
  try { const j = await api('GET', '/jobs/?page_size=1'); jobsCount = j.total || 0; } catch (e) {}
  try { const w = await api('GET', '/warehouse/items'); whCount = (w.items || []).length; } catch (e) {}
  try { const r = await api('GET', '/partners/referrals'); if (r.referrals) { referralsPending = r.referrals.filter(x => x.payout_status === 'pending').length; } } catch (e) {}
  try { const p = await api('GET', '/admin/payments'); if (p.payments) { paymentsSum = p.payments.filter(x => x.status === 'paid').reduce((s, x) => s + x.amount, 0); } } catch (e) {}
  document.getElementById('content').innerHTML =
    '<div class="stats">' +
    '<div class="stat"><div class="num blue">' + jobsCount + '</div><div class="lbl">Наряды</div></div>' +
    '<div class="stat"><div class="num amber">' + whCount + '</div><div class="lbl">Позиций на складе</div></div>' +
    '<div class="stat"><div class="num amber">' + referralsPending + '</div><div class="lbl">Выплат ожидает</div></div>' +
    '<div class="stat"><div class="num green">' + paymentsSum.toFixed(0) + ' ₽</div><div class="lbl">Оплачено</div></div>' +
    '</div>' +
    '<div class="panel"><h2>Последние наряды</h2><table><thead><tr><th>Адрес</th><th>Статус</th><th>Создан</th></tr></thead><tbody id="recentJobs"></tbody></table></div>';
  try {
    const j = await api('GET', '/jobs/?page_size=5');
    const tb = document.getElementById('recentJobs');
    tb.innerHTML = (j.items || []).map(item =>
      '<tr><td>' + (item.address || '—') + '</td><td><span class="badge ' + (item.status || 'new') + '">' + (item.status || 'new') + '</span></td><td>' + (item.created_at ? new Date(item.created_at*1000).toLocaleString('ru-RU') : '—') + '</td></tr>'
    ).join('') || '<tr><td colspan="3" class="loading">Нет нарядов</td></tr>';
  } catch (e) {}
}

async function loadJobs() {
  let html = '<div class="panel"><h2>Наряды</h2><table><thead><tr><th>Адрес</th><th>Статус</th><th>Создан</th></tr></thead><tbody>';
  try {
    const j = await api('GET', '/jobs/?page_size=50');
    html += (j.items || []).map(item =>
      '<tr><td>' + (item.address || '—') + '</td><td><span class="badge ' + (item.status || 'new') + '">' + (item.status || 'new') + '</span></td><td>' + (item.created_at ? new Date(item.created_at*1000).toLocaleString('ru-RU') : '—') + '</td></tr>'
    ).join('') || '<tr><td colspan="3" class="loading">Нет нарядов</td></tr>';
  } catch (e) { html += '<tr><td colspan="3">' + e.message + '</td></tr>'; }
  document.getElementById('content').innerHTML = html + '</tbody></table></div>';
}

async function loadWarehouse() {
  let html = '<div class="panel"><h2>Склад (моя организация)</h2><table><thead><tr><th>Наименование</th><th>Ед.</th><th>Остаток</th><th>Категория</th></tr></thead><tbody>';
  try {
    const w = await api('GET', '/warehouse/items');
    html += (w.items || []).map(item =>
      '<tr><td>' + (item.name || '—') + '</td><td>' + (item.unit || '—') + '</td><td>' + item.quantity + '</td><td>' + (item.category || '—') + '</td></tr>'
    ).join('') || '<tr><td colspan="4" class="loading">Склад пуст</td></tr>';
  } catch (e) { html += '<tr><td colspan="4">' + e.message + '</td></tr>'; }
  document.getElementById('content').innerHTML = html + '</tbody></table></div>';
}

async function loadReferrals() {
  let html = '<div class="panel"><h2>Партнёры и рефералы</h2><table><thead><tr><th>Партнёр</th><th>Подписка</th><th>Сумма</th><th>Статус</th></tr></thead><tbody>';
  try {
    const r = await api('GET', '/partners/referrals');
    html += (r.referrals || []).map(ref =>
      '<tr><td>' + (ref.partner_name || '—') + '</td><td>' + (ref.subscription_id || '—') + '</td><td>' + (ref.amount != null ? ref.amount.toFixed(2) + ' ₽' : '—') + '</td><td><span class="badge ' + (ref.payout_status || 'pending') + '">' + (ref.payout_status || 'pending') + '</span></td></tr>'
    ).join('') || '<tr><td colspan="4" class="loading">Нет рефералов</td></tr>';
  } catch (e) { html += '<tr><td colspan="4">' + e.message + '</td></tr>'; }
  document.getElementById('content').innerHTML = html + '</tbody></table></div>';
}

async function loadPayments() {
  let html = '<div class="panel"><h2>Платежи</h2><table><thead><tr><th>ID</th><th>Сумма</th><th>Статус</th><th>Время</th></tr></thead><tbody>';
  try {
    const p = await api('GET', '/admin/payments');
    html += (p.payments || []).map(x =>
      '<tr><td>' + (x.payment_id || '—') + '</td><td>' + x.amount.toFixed(2) + ' ₽</td><td><span class="badge ' + (x.status || '') + '">' + (x.status || '—') + '</span></td><td>' + (x.paid_at ? new Date(x.paid_at).toLocaleString('ru-RU') : '—') + '</td></tr>'
    ).join('') || '<tr><td colspan="4" class="loading">Нет платежей</td></tr>';
  } catch (e) { html += '<tr><td colspan="4">' + e.message + '</td></tr>'; }
  document.getElementById('content').innerHTML = html + '</tbody></table></div>';
}

async function loadPlanner() {
  let html = '<div class="panel"><h2>Планировщик нарядов</h2>';
  html += '<p style="color:var(--muted);font-size:13px;margin-bottom:12px">Создайте наряд для бригады. Поле «Адрес» — обязательное.</p>';
  html += '<div style="display:flex;flex-wrap:wrap;gap:8px;margin-bottom:8px">';
  html += '<input id="plannerAddress" placeholder="Адрес объекта *" style="min-width:220px">';
  html += '<input id="plannerDesc" placeholder="Вид работ (напр. Монтаж ВОЛС)" style="min-width:220px">';
  html += '</div>';
  html += '<button onclick="createJob()">➕ Создать наряд</button>';
  html += '<div id="plannerMsg" style="margin-top:8px;color:var(--amber)"></div>';
  html += '<h2 style="margin-top:20px">Существующие наряды</h2><table><thead><tr><th>Адрес</th><th>Работы</th><th>Статус</th><th>Создан</th></tr></thead><tbody>';
  try {
    const j = await api('GET', '/jobs/?page_size=50');
    html += (j.items || []).map(item =>
      '<tr><td>' + (item.address || '—') + '</td><td>' + (item.description || '—') + '</td><td><span class="badge ' + (item.status || 'new') + '">' + (item.status || 'new') + '</span></td><td>' + (item.created_at ? new Date(item.created_at*1000).toLocaleString('ru-RU') : '—') + '</td></tr>'
    ).join('') || '<tr><td colspan="4" class="loading">Нарядов нет</td></tr>';
  } catch (e) { html += '<tr><td colspan="4">' + e.message + '</td></tr>'; }
  document.getElementById('content').innerHTML = html + '</tbody></table></div>';
}

async function createJob() {
  const address = document.getElementById('plannerAddress').value.trim();
  const desc = document.getElementById('plannerDesc').value.trim();
  if (!address) { document.getElementById('plannerMsg').innerHTML = 'Укажите адрес'; return; }
  const msg = document.getElementById('plannerMsg');
  try {
    await api('POST', '/jobs/', { address, description: desc });
    msg.innerHTML = '✅ Наряд создан';
    document.getElementById('plannerAddress').value = '';
    document.getElementById('plannerDesc').value = '';
    loadPlanner();
  } catch (e) { msg.innerHTML = '❌ ' + e.message; }
}

async function loadDocs() {
  let html = '<div class="panel"><h2>Генерация актов КС-2 / КС-3</h2>';
  html += '<p style="color:var(--muted);font-size:13px;margin-bottom:12px">Выберите наряд и сформируйте исполнительный документ. Формируется текст акта; подпись УКЭП — на этапе интеграции с сертификатом.</p>';
  html += '<select id="docsJob" style="min-width:260px"><option value="">Загрузка нарядов...</option></select>';
  html += ' <button onclick="genKs2()">📄 КС-2</button> <button onclick="genKs3()">📄 КС-3</button>';
  html += '<pre id="docsOutput" style="margin-top:12px;white-space:pre-wrap"></pre>';
  html += '<div id="docsMsg" style="margin-top:8px;color:var(--amber)"></div>';
  document.getElementById('content').innerHTML = html + '</div>';
  try {
    const j = await api('GET', '/jobs/?page_size=50');
    const sel = document.getElementById('docsJob');
    sel.innerHTML = '<option value="">— выберите наряд —</option>' + (j.items || []).map(x =>
      '<option value="' + x.id + '">' + (x.address || x.id) + '</option>'
    ).join('');
  } catch (e) { document.getElementById('docsMsg').innerHTML = '❌ ' + e.message; }
}

async function genKs2() {
  const jobId = document.getElementById('docsJob').value;
  const out = document.getElementById('docsOutput');
  if (!jobId) { out.textContent = 'Выберите наряд'; return; }
  try {
    const r = await api('POST', '/admin/generate-ks2', { job_id: jobId });
    out.textContent = r.ks2_text || '';
  } catch (e) { out.textContent = 'Ошибка: ' + e.message; }
}

async function genKs3() {
  const jobId = document.getElementById('docsJob').value;
  const out = document.getElementById('docsOutput');
  if (!jobId) { out.textContent = 'Выберите наряд'; return; }
  try {
    const r = await api('POST', '/admin/generate-ks3', { job_id: jobId, total_cost: 0 });
    out.textContent = r.ks3_text || '';
  } catch (e) { out.textContent = 'Ошибка: ' + e.message; }
}

async function loadAudit() {
  let html = '<div class="panel"><h2>Журнал аудита (SHA-256 цепочка)</h2>';
  html += '<p style="color:var(--muted);font-size:13px;margin-bottom:12px">Каждая запись связана с предыдущей хешем. Кнопка ниже проверяет целостность всей цепочки.</p>';
  html += '<button onclick="verifyAudit()">🔗 Проверить целостность</button>';
  html += '<div id="auditVerify" style="margin:8px 0"></div>';
  html += '<table><thead><tr><th>Время</th><th>Действие</th><th>Объект</th><th>Детали</th><th>IP</th><th>Хеш (начало)</th></tr></thead><tbody>';
  try {
    const a = await api('GET', '/admin/audit?limit=30');
    html += (a.audit || []).map(x =>
      '<tr><td>' + (x.time || '—') + '</td><td>' + (x.action || '—') + '</td><td>' + (x.entity_id || '—') + '</td><td>' + (x.detail || '—') + '</td><td>' + (x.ip || '—') + '</td><td><code>' + (x.current_hash ? x.current_hash.slice(0,10) + '…' : '—') + '</code></td></tr>'
    ).join('') || '<tr><td colspan="6" class="loading">Журнал пуст</td></tr>';
  } catch (e) { html += '<tr><td colspan="6">' + e.message + '</td></tr>'; }
  document.getElementById('content').innerHTML = html + '</tbody></table></div>';
}

async function verifyAudit() {
  const out = document.getElementById('auditVerify');
  try {
    const v = await api('GET', '/admin/audit/verify');
    out.innerHTML = '<span class="' + (v.valid ? 'badge completed' : 'badge cancelled') + '">' +
      (v.valid ? '✅ Цепочка цела (' + (v.total||0) + ' записей)' : '❌ Нарушена в ' + (v.broken_at||'?')) + '</span>';
  } catch (e) { out.innerHTML = '❌ ' + e.message; }
}

async function loadAlerts() {
  let html = '<div class="panel"><h2>Оповестить бригаду</h2>';
  html += '<p style="color:var(--muted);font-size:13px;margin-bottom:12px">Критическое оповещение приходит на телефоны бригады полным экраном; оно не закрывается без «Подтвердить получение» (диспетчер видит подтверждение в журнале).</p>';
  html += '<input id="alertTitle" placeholder="Заголовок * (напр. Штормовое предупреждение)" style="min-width:260px">';
  html += '<input id="alertMsg" placeholder="Сообщение (напр. Гроза в 14:00 — прекратите работы)" style="min-width:260px">';
  html += '<select id="alertSeverity"><option value="critical">Критический</option><option value="warning">Предупреждение</option><option value="info">Инфо</option></select>';
  html += ' <button onclick="sendAlert()">🚨 Отправить</button>';
  html += '<div id="alertMsgOut" style="margin-top:8px;color:var(--amber)"></div>';
  html += '<h2 style="margin-top:20px">Неподтверждённые</h2><table><thead><tr><th>Заголовок</th><th>Важность</th><th>Статус</th></tr></thead><tbody>';
  try {
    const a = await api('GET', '/alerts');
    html += (a.alerts || []).map(x =>
      '<tr><td>' + (x.title || '—') + '</td><td>' + (x.severity || '—') + '</td><td><span class="badge ' + (x.status || '') + '">' + (x.status || '—') + '</span></td></tr>'
    ).join('') || '<tr><td colspan="3" class="loading">Нет неподтверждённых</td></tr>';
  } catch (e) { html += '<tr><td colspan="3">' + e.message + '</td></tr>'; }
  document.getElementById('content').innerHTML = html + '</tbody></table></div>';
}

async function sendAlert() {
  const title = document.getElementById('alertTitle').value.trim();
  const msg = document.getElementById('alertMsg').value.trim();
  const severity = document.getElementById('alertSeverity').value;
  const out = document.getElementById('alertMsgOut');
  if (!title) { out.innerHTML = 'Укажите заголовок'; return; }
  try {
    await api('POST', '/alerts', { title, message: msg, severity });
    out.innerHTML = '✅ Оповещение отправлено. Бригада увидит его полным экраном.';
    document.getElementById('alertTitle').value = '';
    document.getElementById('alertMsg').value = '';
    loadAlerts();
  } catch (e) { out.innerHTML = '❌ ' + e.message; }
}

async function loadIntegrity() {
  let html = '<div class="panel"><h2>Целостность данных (SHA-256)</h2>';
  html += '<p style="color:var(--muted);font-size:13px;margin-bottom:12px">Журнал действий защищён хеш-цепочкой: каждая запись связана с предыдущей. Изменить или удалить данные задним числом невозможно — проверка это покажет.</p>';
  try {
    const v = await api('GET', '/admin/audit/verify');
    const ok = v.valid === true;
    html += '<div class="stat" style="max-width:300px">';
    html += '<div class="num ' + (ok ? 'green' : 'red') + '">' + (ok ? '✅ Целостность подтверждена' : '❌ Цепочка нарушена') + '</div>';
    html += '<div class="lbl">Записей в журнале: ' + (v.total || 0) + '</div></div>';
  } catch (e) {
    html += '<div class="stat" style="max-width:300px"><div class="num amber">Нет доступа</div><div class="lbl">' + e.message + '</div></div>';
  }
  document.getElementById('content').innerHTML = html + '</div>';
}

function openPresentation() {
  window.open('/presentation', '_blank');
}

// Авто-обновление: подхватываем изменения с телефона каждые 8 секунд.
setInterval(function() {
  const active = document.activeElement;
  const editing = active && (active.tagName === 'INPUT' || active.tagName === 'SELECT');
  if (token() && !editing) loadPage(currentPage);
}, 8000);

// Начальная загрузка
if (token()) {
  document.getElementById('loginForm').style.display = 'none';
  document.getElementById('mainUI').style.display = 'flex';
  document.getElementById('userInfo').textContent = username();
  loadPage('stats');
}
</script>
</body>
</html>`