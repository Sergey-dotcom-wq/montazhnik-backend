package web

// AdminHTML — встроенная веб-страница админки (двуязычная: RU/EN)
const AdminHTML = `<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title id="pageTitle">Монтажник связи — Админка</title>
<style>
* { margin:0; padding:0; box-sizing:border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0f172a; color: #e2e8f0; padding: 20px; }
h1 { color: #38bdf8; margin-bottom: 20px; display: inline-block; }
h2 { color: #94a3b8; margin: 20px 0 10px; font-size: 1.1em; text-transform: uppercase; letter-spacing: 0.05em; }
.card { background: #1e293b; border-radius: 8px; padding: 16px; margin-bottom: 16px; }
table { width: 100%; border-collapse: collapse; font-size: 0.9em; }
th, td { padding: 8px 12px; text-align: left; border-bottom: 1px solid #334155; }
th { color: #94a3b8; font-weight: 600; }
tr:hover { background: #334155; }
.status-new { color: #facc15; }
.status-assigned { color: #facc15; }
.status-in_progress { color: #38bdf8; }
.status-completed { color: #4ade80; }
.status-cancelled { color: #f87171; }
.nav { display: flex; gap: 12px; margin-bottom: 20px; flex-wrap: wrap; }
.nav a { color: #38bdf8; text-decoration: none; padding: 6px 14px; border: 1px solid #334155; border-radius: 6px; cursor: pointer; }
.nav a:hover { background: #334155; }
pre { background: #0f172a; padding: 8px; border-radius: 4px; overflow-x: auto; font-size: 0.8em; }
#content { margin-top: 16px; }
select, input { background: #0f172a; color: #e2e8f0; border: 1px solid #334155; padding: 6px 10px; border-radius: 4px; margin: 0 4px; }
button { background: #38bdf8; color: #0f172a; border: none; padding: 6px 14px; border-radius: 4px; cursor: pointer; font-weight: 600; }
button:hover { background: #7dd3fc; }
button.danger { background: #ef4444; }
button.danger:hover { background: #f87171; }
.loading { color: #64748b; text-align: center; padding: 20px; }
#loginForm { max-width: 360px; margin: 40px auto; }
#loginForm input { display: block; width: 100%; margin: 8px 0; padding: 10px; border-radius: 6px; }
#loginForm button { width: 100%; margin-top: 8px; padding: 10px; }
#loginError { color: #f87171; margin-top: 8px; display: none; }
#userInfo { display: none; align-items: center; gap: 8px; margin-bottom: 8px; color: #94a3b8; }
.lang-switch { float: right; margin-top: 8px; }
.lang-switch button { background: #1e293b; color: #94a3b8; border: 1px solid #334155; }
.lang-switch button.active { background: #38bdf8; color: #0f172a; }
@media(max-width:600px){ table, thead, tbody, th, td, tr { display: block; } th { display: none; } td { padding: 6px 8px; border: none; } td:before { content: attr(data-label); font-weight: 600; color: #94a3b8; display: inline-block; width: 100px; } }
</style>
</head>
<body>
<div class="lang-switch">
  <button id="langRu" onclick="setLang('ru')">RU</button>
  <button id="langEn" onclick="setLang('en')">EN</button>
</div>
<h1 id="appTitle">🔧 Монтажник связи</h1>

<div id="loginForm" class="card">
  <h2 id="loginTitle">Вход в админку</h2>
  <input type="text" id="loginPhone" placeholder="Логин (лат., до 10)">
  <input type="password" id="loginPassword" placeholder="Пароль">
  <div id="loginError"></div>
  <button onclick="login()" id="loginBtn">Войти</button>
</div>

<div id="app" style="display:none">
  <div id="userInfo">
    <span id="userName"></span>
    <button class="danger" onclick="logout()" id="logoutBtn">Выйти</button>
  </div>
  <div class="nav">
    <a onclick="loadPage('jobs')" data-page="jobs" id="navJobs">📋 Наряды</a>
    <a onclick="loadPage('materials')" data-page="materials" id="navMaterials">📦 Материалы</a>
    <a onclick="loadPage('chain')" data-page="chain" id="navChain">🔗 Цепочка</a>
    <a onclick="loadPage('partners')" data-page="partners" id="navPartners" style="display:none">🤝 Партнёры</a>
    <a onclick="loadPage('referrals')" data-page="referrals" id="navReferrals" style="display:none">📈 Рефералы</a>
  </div>
  <div class="card" id="filters" style="display:none">
    <span id="filterContent"></span>
    <button onclick="loadPage(currentPage)" id="refreshBtn">Обновить</button>
  </div>
  <div id="content">
    <div class="loading" id="loadingText">Загрузка...</div>
  </div>
</div>

<script>
const API = '/api/v1';
let currentPage = 'jobs';
let currentRole = localStorage.getItem('admin_role') || '';
let lang = localStorage.getItem('admin_lang') || 'ru';

const I18N = {
  ru: {
    appTitle: '🔧 Монтажник связи',
    pageTitle: 'Монтажник связи — Админка',
    loginTitle: 'Вход в админку',
    loginPhone: 'Логин (лат., до 10)',
    loginPassword: 'Пароль',
    loginBtn: 'Войти',
    logoutBtn: 'Выйти',
    navJobs: '📋 Наряды',
    navMaterials: '📦 Материалы',
    navChain: '🔗 Цепочка',
    refresh: 'Обновить',
    loading: 'Загрузка...',
    user: 'Пользователь',
    filter: 'Фильтр:',
    errAuth: 'Требуется авторизация',
    errLogin: 'Неверный логин или пароль',
    errGeneric: 'Ошибка',
    jobsTitle: 'Наряды (всего: %s)',
    allStatuses: 'Все статусы',
    stNew: 'Новый',
    stAssigned: 'Назначен',
    stInProgress: 'В работе',
    stCompleted: 'Завершён',
    stCancelled: 'Отменён',
    tAddress: 'Адрес',
    tStatus: 'Статус',
    tCreated: 'Создан',
    tActions: 'Действия',
    chainBtn: 'Цепочка',
    chainTitle: 'Цепочка событий:',
    backTo: '← К списку',
    tEventType: 'Тип',
    tTimeUtc: 'Время (UTC)',
    chainVerify: 'Проверка целостности:',
    chainValid: '✅ Цепочка валидна',
    chainBroken: '❌ Цепочка нарушена',
    materialsTitle: 'Справочник материалов',
    allCats: 'Все',
    catCable: 'Кабель',
    catClosure: 'Муфта',
    catConsumable: 'Расходник',
    addMaterial: '+ Добавить',
    tSku: 'Артикул',
    tName: 'Наименование',
    tUnit: 'Ед.',
    tPrice: 'Цена',
    tVat: 'НДС',
    tCategory: 'Категория',
    mName: 'Наименование:',
    mSku: 'Артикул (SKU):',
    mUnit: 'Единица измерения (шт/м/бухта):',
    mPrice: 'Цена (₽):',
    chainIdPrompt: 'Введите ID наряда:',
    loadingChain: 'Загрузка цепочки...',
    notFound: 'Объект не найден или сервер недоступен',
    navPartners: '🤝 Партнёры',
    navReferrals: '📈 Рефералы',
    partnersTitle: 'Партнёры',
    referralsTitle: 'Рефералы',
    tInn: 'ИНН',
    tPartnerName: 'Партнёр',
    tReferralCode: 'Код',
    tPayoutPercent: '%',
    tBankAccount: 'Счёт',
    tAmount: 'Сумма',
    tPayoutStatus: 'Статус выплаты',
    psPending: 'Ожидает',
    psPaid: 'Выплачено',
    psFailed: 'Ошибка',
    tSubscription: 'Подписка'
  },
en: {
    appTitle: '🔧 Installer HQ',
    pageTitle: 'Installer HQ — Admin console',
    loginTitle: 'Admin login',
    loginPhone: 'Login (latin, max 10)',
    loginPassword: 'Password',
    loginBtn: 'Sign in',
    logoutBtn: 'Log out',
    navJobs: '📋 Work orders',
    navMaterials: '📦 Materials',
    navChain: '🔗 Event chain',
    refresh: 'Refresh',
    loading: 'Loading...',
    user: 'User',
    errAuth: 'Authorization required',
    errLogin: 'Invalid login or password',
    errGeneric: 'Error',
    jobsTitle: 'Work orders (total: %s)',
    allStatuses: 'All statuses',
    stNew: 'New',
    stAssigned: 'Assigned',
    stInProgress: 'In progress',
    stCompleted: 'Completed',
    stCancelled: 'Cancelled',
    tAddress: 'Address',
    tStatus: 'Status',
    tCreated: 'Created',
    tActions: 'Actions',
    chainBtn: 'Chain',
    chainTitle: 'Event chain:',
    backTo: '← Back to list',
    tEventType: 'Type',
    tTimeUtc: 'Time (UTC)',
    chainVerify: 'Integrity check:',
    chainValid: '✅ Chain is valid',
    chainBroken: '❌ Chain is broken',
    materialsTitle: 'Materials reference',
    allCats: 'All',
    catCable: 'Cable',
    catClosure: 'Closure',
    catConsumable: 'Consumable',
    addMaterial: '+ Add',
    tSku: 'SKU',
    tName: 'Name',
    tUnit: 'Unit',
    tPrice: 'Price',
    tVat: 'VAT',
    tCategory: 'Category',
    mName: 'Name:',
    mSku: 'SKU:',
    mUnit: 'Unit (pcs/m/coil):',
    mPrice: 'Price (₽):',
    chainIdPrompt: 'Enter work order ID:',
    loadingChain: 'Loading chain...',
    notFound: 'Object not found or server unavailable',
    navPartners: '🤝 Partners',
    navReferrals: '📈 Referrals',
    partnersTitle: 'Partners',
    referralsTitle: 'Referrals',
    tInn: 'INN',
    tPartnerName: 'Partner',
    tReferralCode: 'Code',
    tPayoutPercent: '%',
    tBankAccount: 'Account',
    tAmount: 'Amount',
    tPayoutStatus: 'Payout status',
    psPending: 'Pending',
    psPaid: 'Paid',
    psFailed: 'Failed',
    tSubscription: 'Subscription'
  }
};

// Словарь переводов названий данных (материалы, наряды, склад) для EN.
// Русские названия из БД остаются как есть, в английском интерфейсе
// подставляется перевод.
const DATA_TRANSLATIONS = {
  // Складские позиции
  'Кабель оптический SM': 'Optical cable SM',
  'Кабель оптический OS2 1 км': 'Optical cable OS2 1 km',
  'Муфта OPT-12': 'Closure OPT-12',
  'Муфта оптическая OTS-12': 'Optical closure OTS-12',
  '4G-роутер Teltonika RUT240': '4G router Teltonika RUT240',
  'ONT-терминал': 'ONT terminal',
  'SIM-карта МТС': 'SIM card MTS',
  'Антенна внешняя 12dBi': 'External antenna 12dBi',
  'Гильза КДЗС 60мм': 'KZDS sleeve 60mm',
  'Изолента ПВХ': 'PVC insulation tape',
  'Патч-корд SC/APC 3м': 'Patch cord SC/APC 3m',
  'Репитель VEGATEL AV1-5B': 'Repeater VEGATEL AV1-5B',
  // Единицы измерения
  'шт': 'pcs', 'м': 'm', 'бухта': 'coil', 'км': 'km',
  // Категории
  'оборудование': 'equipment', 'расходник': 'consumable', 'антенна': 'antenna',
  'кабель': 'cable', 'муфта': 'closure',
  // Адреса нарядов
  'г. Москва, ул. Строителей, 12': 'Moscow, Stroitely str. 12',
  'г. Санкт-Петербург, Невский пр., 1': 'Saint Petersburg, Nevsky ave. 1',
  // Типы работ
  'Монтажные работы': 'Installation works',
  'Монтаж ВОЛС': 'Fiber installation',
  'Монтаж СКС': 'Structured cabling installation',
  'Ремонт ВОЛС': 'Fiber repair'
};

function translateData(s) {
  if (!s) return s;
  if (lang !== 'en') return s;
  return DATA_TRANSLATIONS[s] || s;
}

function t(key) {
  const s = (I18N[lang] || I18N.ru)[key] || key;
  return s;
}

function fmt(key, arg) {
  return t(key).replace('%s', arg);
}

function setLang(l) {
  lang = l;
  localStorage.setItem('admin_lang', l);
  document.getElementById('langRu').className = l === 'ru' ? 'active' : '';
  document.getElementById('langEn').className = l === 'en' ? 'active' : '';
  applyStatic();
  if (typeof token() === 'string' && token()) {
    loadPage(currentPage);
  }
}

function applyStatic() {
  document.getElementById('appTitle').textContent = t('appTitle');
  document.getElementById('pageTitle').textContent = t('pageTitle');
  document.title = t('pageTitle');
  document.getElementById('loginTitle').textContent = t('loginTitle');
  document.getElementById('loginPhone').placeholder = t('loginPhone');
  document.getElementById('loginPassword').placeholder = t('loginPassword');
  document.getElementById('loginBtn').textContent = t('loginBtn');
  document.getElementById('logoutBtn').textContent = t('logoutBtn');
  document.getElementById('navJobs').textContent = t('navJobs');
  document.getElementById('navMaterials').textContent = t('navMaterials');
  document.getElementById('navChain').textContent = t('navChain');
  document.getElementById('navPartners').textContent = t('navPartners');
  document.getElementById('navReferrals').textContent = t('navReferrals');
  document.getElementById('refreshBtn').textContent = t('refresh');
  document.getElementById('loadingText').textContent = t('loading');
  applyRoleNav();
}

// Показываем только те разделы, что доступны текущей роли.
function applyRoleNav() {
  const role = currentRole || '';
  const showJobs = role === '' || role === 'admin' || role === 'dispatcher';
  const showMaterials = role === '' || role === 'admin' || role === 'dispatcher' || role === 'accountant';
  const showPartners = role === '' || role === 'admin' || role === 'accountant' || role === 'partner';
  const showReferrals = role === '' || role === 'admin' || role === 'accountant' || role === 'partner';
  document.getElementById('navJobs').style.display = showJobs ? '' : 'none';
  document.getElementById('navMaterials').style.display = showMaterials ? '' : 'none';
  document.getElementById('navChain').style.display = showJobs ? '' : 'none';
  document.getElementById('navPartners').style.display = showPartners ? '' : 'none';
  document.getElementById('navReferrals').style.display = showReferrals ? '' : 'none';
  // Раздел по умолчанию под роль
  if (!showJobs && showMaterials) currentPage = 'materials';
  else if (!showJobs && !showMaterials && showReferrals) currentPage = 'referrals';
}

function token() { return localStorage.getItem('admin_token'); }

async function api(method, path, body) {
  const opts = { method, headers: { 'Content-Type': 'application/json' } };
  const tok = token();
  if (tok) opts.headers['Authorization'] = 'Bearer ' + tok;
  if (body) opts.body = JSON.stringify(body);
  const r = await fetch(API + path, opts);
  if (r.status === 401) { logout(); throw new Error(t('errAuth')); }
  if (!r.ok) { const e = await r.text(); throw new Error(r.status + ': ' + e); }
  return r.headers.get('content-type')?.includes('json') ? r.json() : r.text();
}

async function login() {
  const username = document.getElementById('loginPhone').value;
  const password = document.getElementById('loginPassword').value;
  document.getElementById('loginError').style.display = 'none';
  try {
    const r = await fetch(API + '/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password })
    });
    if (!r.ok) { throw new Error(t('errLogin')); }
    const data = await r.json();
    localStorage.setItem('admin_token', data.token);
    currentRole = data.user?.role || '';
    localStorage.setItem('admin_role', currentRole);
    document.getElementById('loginForm').style.display = 'none';
    document.getElementById('app').style.display = 'block';
    document.getElementById('userName').textContent = t('user') + ': ' + (data.user?.full_name || data.user?.username || username) + (currentRole ? ' · ' + currentRole : '');
    applyRoleNav();
    loadPage(currentPage);
  } catch(e) {
    document.getElementById('loginError').textContent = t('errGeneric') + ': ' + e.message;
    document.getElementById('loginError').style.display = 'block';
  }
}

function logout() {
  localStorage.removeItem('admin_token');
  localStorage.removeItem('admin_role');
  currentRole = '';
  document.getElementById('app').style.display = 'none';
  document.getElementById('loginForm').style.display = 'block';
  document.getElementById('loginPhone').value = '';
  document.getElementById('loginPassword').value = '';
}

async function loadPage(page) {
  currentPage = page;
  document.getElementById('filters').style.display = page === 'jobs' || page === 'materials' ? 'block' : 'none';
  document.getElementById('content').innerHTML = '<div class="loading">' + t('loading') + '</div>';
  try {
    if (page === 'jobs') await loadJobs();
    else if (page === 'materials') await loadMaterials();
    else if (page === 'chain') await loadChain();
    else if (page === 'partners') await loadPartners();
    else if (page === 'referrals') await loadReferrals();
  } catch(e) {
    document.getElementById('content').innerHTML = '<div class="loading" style="color:#f87171">' + t('errGeneric') + ': ' + e.message + '</div>';
  }
}

async function loadPartners() {
  const data = await api('GET', '/partners');
  let html = '<h2>' + t('partnersTitle') + '</h2>';
  html += '<table><thead><tr><th>' + t('tPartnerName') + '</th><th>' + t('tInn') + '</th><th>' + t('tReferralCode') + '</th><th>' + t('tPayoutPercent') + '</th><th>' + t('tBankAccount') + '</th><th>' + t('tStatus') + '</th></tr></thead><tbody>';
  for (const p of data.partners) {
    html += '<tr><td data-label="' + t('tPartnerName') + '">' + translateData(p.full_name || '—') + '</td>';
    html += '<td data-label="' + t('tInn') + '">' + (p.inn || '—') + '</td>';
    html += '<td data-label="' + t('tReferralCode') + '"><code>' + (p.referral_code || '—') + '</code></td>';
    html += '<td data-label="' + t('tPayoutPercent') + '">' + (p.payout_percent || 0) + '%</td>';
    html += '<td data-label="' + t('tBankAccount') + '">' + (p.bank_account || '—') + '</td>';
    html += '<td data-label="' + t('tStatus') + '">' + (p.status || '—') + '</td></tr>';
  }
  html += '</tbody></table>';
  document.getElementById('content').innerHTML = html;
}

async function loadReferrals() {
  // Сервер сам определяет, какие рефералы показывать по роли
  // (partner — только свои, admin/accountant — все).
  const data = await api('GET', '/partners/referrals');
  let html = '<h2>' + t('referralsTitle') + '</h2>';
  html += '<table><thead><tr><th>' + t('tPartnerName') + '</th><th>' + t('tSubscription') + '</th><th>' + t('tAmount') + '</th><th>' + t('tPayoutStatus') + '</th></tr></thead><tbody>';
  for (const r of data.referrals) {
    const ps = r.payout_status === 'paid' ? t('psPaid') : (r.payout_status === 'failed' ? t('psFailed') : t('psPending'));
    html += '<tr><td data-label="' + t('tPartnerName') + '">' + translateData(r.partner_name || '—') + '</td>';
    html += '<td data-label="' + t('tSubscription') + '">' + (r.subscription_id || '—') + '</td>';
    html += '<td data-label="' + t('tAmount') + '">' + (r.amount != null ? r.amount.toFixed(2) + '₽' : '—') + '</td>';
    html += '<td data-label="' + t('tPayoutStatus') + '">' + ps + '</td></tr>';
  }
  html += '</tbody></table>';
  document.getElementById('content').innerHTML = html;
}

function statusLabel(st) {
  const map = { new: 'stNew', assigned: 'stAssigned', in_progress: 'stInProgress', completed: 'stCompleted', cancelled: 'stCancelled' };
  const key = map[st] || 'stNew';
  return t(key);
}

async function loadJobs() {
  const status = document.getElementById('filter-status')?.value || '';
  const url = '/jobs/?page_size=50' + (status ? '&status=' + status : '');
  const data = await api('GET', url);
  let html = '<h2>' + fmt('jobsTitle', data.total) + '</h2>';
  html += '<div id="filterContent">' + t('filter') + ' <select id="filter-status" onchange="loadPage(\'jobs\')">';
  html += '<option value="">' + t('allStatuses') + '</option>';
  html += '<option value="new"' + (status==='new'?' selected':'') + '>' + t('stNew') + '</option>';
  html += '<option value="assigned"' + (status==='assigned'?' selected':'') + '>' + t('stAssigned') + '</option>';
  html += '<option value="in_progress"' + (status==='in_progress'?' selected':'') + '>' + t('stInProgress') + '</option>';
  html += '<option value="completed"' + (status==='completed'?' selected':'') + '>' + t('stCompleted') + '</option>';
  html += '<option value="cancelled"' + (status==='cancelled'?' selected':'') + '>' + t('stCancelled') + '</option></select></div>';
  html += '<table><thead><tr><th>ID</th><th>' + t('tAddress') + '</th><th>' + t('tStatus') + '</th><th>' + t('tCreated') + '</th><th>' + t('tActions') + '</th></tr></thead><tbody>';
  for (const j of data.items) {
    html += '<tr><td data-label="ID"><code>' + (j.id ? j.id.substr(0,8) : '—') + '...</code></td>';
    html += '<td data-label="' + t('tAddress') + '">' + translateData(j.address || '—') + '</td>';
    html += '<td data-label="' + t('tStatus') + '"><span class="status-' + (j.status || 'new') + '">' + statusLabel(j.status || 'new') + '</span></td>';
    html += '<td data-label="' + t('tCreated') + '">' + (j.created_at ? new Date(j.created_at*1000).toLocaleString(lang === 'ru' ? 'ru-RU' : 'en-GB') : '—') + '</td>';
    html += '<td data-label="' + t('tActions') + '"><button onclick="viewJob(\'' + (j.id || '') + '\')">' + t('chainBtn') + '</button></td></tr>';
  }
  html += '</tbody></table>';
  document.getElementById('content').innerHTML = html;
}

async function viewJob(id) {
  currentPage = 'chain';
  document.getElementById('filters').style.display = 'none';
  document.getElementById('content').innerHTML = '<div class="loading">' + t('loadingChain') + '</div>';
  const events = await api('GET', '/events/' + id + '/chain');
  let html = '<h2>' + t('chainTitle') + ' <code>' + (id ? id.substr(0,8) : '—') + '...</code></h2>';
  html += '<button onclick="loadPage(\'jobs\')" style="margin-bottom:12px">' + t('backTo') + '</button>';
  html += '<table><thead><tr><th>#</th><th>' + t('tEventType') + '</th><th>Payload</th><th>Previous Hash</th><th>Current Hash</th><th>' + t('tTimeUtc') + '</th></tr></thead><tbody>';
  for (let i = 0; i < events.length; i++) {
    const e = events[i];
    const ts = new Date(e.CreatedAt).toLocaleString(lang === 'ru' ? 'ru-RU' : 'en-GB');
    html += '<tr><td data-label="#">' + (i+1) + '</td>';
    html += '<td data-label="' + t('tEventType') + '">' + e.EventType + '</td>';
    html += '<td data-label="Payload"><pre>' + JSON.stringify(JSON.parse(e.Payload || '{}'), null, 1) + '</pre></td>';
    html += '<td data-label="Prev"><code>' + (e.PreviousHash ? e.PreviousHash.substr(0,12) + '...' : '—') + '</code></td>';
    html += '<td data-label="Curr"><code>' + e.CurrentHash.substr(0,12) + '...</code></td>';
    html += '<td data-label="' + t('tTimeUtc') + '">' + ts + '</td></tr>';
  }
  html += '</tbody></table>';
  const verify = await api('GET', '/events/' + id + '/verify');
  html += '<div style="margin-top:12px;padding:8px;background:#0f172a;border-radius:4px">';
  html += t('chainVerify') + ' <strong>' + (verify.is_valid ? t('chainValid') : t('chainBroken')) + '</strong></div>';
  document.getElementById('content').innerHTML = html;
}

async function loadMaterials() {
  const cat = document.getElementById('filter-cat')?.value || '';
  const url = '/materials/' + (cat ? '?category=' + cat : '');
  const items = await api('GET', url);
  let html = '<h2>' + t('materialsTitle') + '</h2>';
  html += '<div id="filterContent">' + t('tCategory') + ': <select id="filter-cat" onchange="loadPage(\'materials\')">';
  html += '<option value="">' + t('allCats') + '</option><option value="кабель"' + (cat==='кабель'?' selected':'') + '>' + t('catCable') + '</option>';
  html += '<option value="муфта"' + (cat==='муфта'?' selected':'') + '>' + t('catClosure') + '</option>';
  html += '<option value="расходник"' + (cat==='расходник'?' selected':'') + '>' + t('catConsumable') + '</option></select></div>';
  html += '<button onclick="showAddMaterial()" style="margin-left:10px">' + t('addMaterial') + '</button>';
  html += '<table><thead><tr><th>' + t('tSku') + '</th><th>' + t('tName') + '</th><th>' + t('tUnit') + '</th><th>' + t('tPrice') + '</th><th>' + t('tVat') + '</th><th>' + t('tCategory') + '</th></tr></thead><tbody>';
  for (const m of items) {
    html += '<tr><td data-label="' + t('tSku') + '"><code>' + (m.SKU || '—') + '</code></td>';
    html += '<td data-label="' + t('tName') + '">' + (m.Name || '—') + '</td>';
    html += '<td data-label="' + t('tUnit') + '">' + (m.Unit || '—') + '</td>';
    html += '<td data-label="' + t('tPrice') + '">' + (m.Price != null ? m.Price.toFixed(2) + '₽' : '—') + '</td>';
    html += '<td data-label="' + t('tVat') + '">' + (m.VAT || 0) + '%</td>';
    html += '<td data-label="' + t('tCategory') + '">' + (m.Category || '—') + '</td></tr>';
  }
  html += '</tbody></table>';
  document.getElementById('content').innerHTML = html;
}

function showAddMaterial() {
  const name = prompt(t('mName'));
  if (!name) return;
  const sku = prompt(t('mSku'));
  const unit = prompt(t('mUnit'));
  const price = parseFloat(prompt(t('mPrice')));
  api('POST', '/materials/', { sku, name, unit, price, vat: 20 }).then(() => loadPage('materials')).catch(e => alert(t('errGeneric') + ': ' + e.message));
}

async function loadChain() {
  const id = prompt(t('chainIdPrompt'));
  if (!id) { document.getElementById('content').innerHTML = '<div class="loading">' + t('loading') + '</div>'; return; }
  await viewJob(id);
}

// Initial load
applyStatic();
setLang(lang);
if (token()) {
  document.getElementById('loginForm').style.display = 'none';
  document.getElementById('app').style.display = 'block';
  loadPage('jobs');
}

// Auto-refresh: подхватываем изменения с телефона каждые 8 секунд.
// Не обновляем на странице цепочки (viewJob) и пока пользователь
// редактирует фильтры или что-то вводит на странице материалов.
setInterval(function() {
  const active = document.activeElement;
  const editing = active && (active.tagName === 'INPUT' || active.tagName === 'SELECT');
  if (!editing && currentPage !== 'chain') {
    loadPage(currentPage);
  }
}, 8000);
</script>
</body>
</html>`