package web

// LandingHTML — публичный лендинг «Монтажник Связи» (фуршетный сайт).
// Презентация, тарифы, партнёрская программа, кнопки подписки и регистрации.
// {{REF}} подставляется сервером из реферального кода в URL (?ref=КОД).
const LandingHTML = `<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Монтажник Связи — платформа для монтажных бригад</title>
<style>
:root {
  --bg: #0d1117; --panel: #161b22; --border: #30363d;
  --text: #e6edf3; --muted: #8b949e;
  --accent: #58a6ff; --green: #3fb950; --gold: #d29922; --red: #f85149;
}
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, "Segoe UI", Roboto, Arial, sans-serif; background: var(--bg); color: var(--text); line-height: 1.6; }
a { color: var(--accent); text-decoration: none; }
.container { max-width: 1100px; margin: 0 auto; padding: 0 24px; }

/* Шапка */
header { position: sticky; top: 0; z-index: 20; background: rgba(13,17,23,.92); backdrop-filter: blur(8px); border-bottom: 1px solid var(--border); }
.header-inner { display: flex; align-items: center; gap: 16px; padding: 14px 0; }
.logo { font-size: 20px; font-weight: 800; letter-spacing: .5px; }
.logo .accent { color: var(--accent); }
nav { margin-left: auto; display: flex; gap: 20px; }
nav a { color: var(--muted); font-size: 14px; }
nav a:hover { color: var(--text); }
.btn { display: inline-block; background: var(--accent); color: #0d1117; font-weight: 700; padding: 12px 24px; border-radius: 8px; font-size: 15px; border: none; cursor: pointer; }
.btn:hover { filter: brightness(1.15); }
.btn.outline { background: transparent; color: var(--accent); border: 1px solid var(--accent); }
.btn.large { padding: 16px 32px; font-size: 17px; }

/* Герой */
.hero { text-align: center; padding: 80px 0 60px; }
.hero h1 { font-size: clamp(2.2rem, 6vw, 4.4rem); line-height: 1.1; margin-bottom: 20px; }
.hero .lead { font-size: clamp(1.1rem, 2.6vw, 1.6rem); color: var(--muted); max-width: 800px; margin: 0 auto 32px; }
.hero .actions { display: flex; gap: 16px; justify-content: center; flex-wrap: wrap; }
.ref-badge { display: inline-block; background: rgba(210,153,34,.12); color: var(--gold); border: 1px solid rgba(210,153,34,.4); border-radius: 20px; padding: 6px 16px; font-size: 13px; margin-bottom: 20px; }

/* Секции */
.section { padding: 60px 0; border-top: 1px solid var(--border); }
.section h2 { font-size: clamp(1.6rem, 4vw, 2.4rem); text-align: center; margin-bottom: 12px; }
.section .sub { text-align: center; color: var(--muted); margin-bottom: 40px; }
.grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 16px; }
.card { background: var(--panel); border: 1px solid var(--border); border-radius: 14px; padding: 24px; }
.card .icon { font-size: 2rem; margin-bottom: 12px; }
.card h3 { font-size: 1.15rem; margin-bottom: 8px; }
.card p { color: var(--muted); font-size: .95rem; }
.card .price { font-size: 1.8rem; font-weight: 800; margin: 12px 0 4px; }
.card .price .old { font-size: 1.1rem; color: var(--muted); text-decoration: line-through; font-weight: 400; }
.card ul { list-style: none; margin-top: 12px; }
.card ul li { font-size: .9rem; color: var(--muted); padding: 4px 0; border-bottom: 1px solid var(--border); }
.card ul li:last-child { border-bottom: none; }
.card ul li::before { content: "✓ "; color: var(--green); }

/* Тарифы */
.tiers { display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 20px; margin-top: 10px; }
.tier { background: var(--panel); border: 1px solid var(--border); border-radius: 16px; padding: 28px; text-align: center; }
.tier.featured { border: 2px solid var(--accent); }
.tier .name { font-size: 1.1rem; color: var(--muted); text-transform: uppercase; letter-spacing: .06em; }
.tier .amount { font-size: 2.6rem; font-weight: 800; margin: 8px 0; }
.tier .per { color: var(--muted); font-size: .9rem; }
.tier .features { list-style: none; margin: 20px 0; text-align: left; }
.tier .features li { font-size: .95rem; color: var(--text); padding: 6px 0; border-bottom: 1px solid var(--border); }
.tier .features li::before { content: "✓ "; color: var(--green); }

/* Партнёрская программа */
.partner { background: linear-gradient(135deg, rgba(88,166,255,.08), rgba(63,185,80,.08)); border: 1px solid var(--accent); border-radius: 16px; padding: 32px; text-align: center; margin-top: 20px; }
.partner .big { font-size: 2rem; font-weight: 800; color: var(--green); }
.partner p { color: var(--muted); margin-top: 8px; }

/* Call to action */
.cta { text-align: center; padding: 80px 0; }
.cta h2 { font-size: clamp(1.8rem, 4vw, 2.6rem); margin-bottom: 12px; }
.cta p { color: var(--muted); margin-bottom: 28px; }

/* Презентация на сайте */
.deck-link { display: block; margin: 16px auto 0; }

footer { border-top: 1px solid var(--border); padding: 28px 0; text-align: center; color: var(--muted); font-size: 13px; }
@media (max-width: 700px) { nav { display: none; } }
</style>
</head>
<body>
<header>
  <div class="container header-inner">
    <div class="logo"><span class="accent">Монтажник</span> Связи</div>
    <nav>
      <a href="#features">Возможности</a>
      <a href="#pricing">Тарифы</a>
      <a href="#partner">Партнёрам</a>
      <a href="/presentation">Презентация</a>
    </nav>
  </div>
</header>

<section class="hero">
  <div class="container">
    <div class="ref-badge" id="refBadge" style="display:none">🎁 Ваш реферальный код: <b id="refCode"></b> — скидка 30% на первый месяц</div>
    <h1>Склад, бригады и деньги —<br>под полным контролем</h1>
    <p class="lead">Платформа для монтажных бригад и подрядчиков. Приложение для монтажников, сайт для руководителя. Данные нельзя изменить задним числом.</p>
    <div class="actions">
      <a class="btn large" href="#" id="subscribeBtn">Оформить подписку — 3 000 ₽/мес</a>
      <a class="btn outline large" href="#features">Узнать больше</a>
    </div>
  </div>
</section>

<section class="section" id="features">
  <div class="container">
    <h2>Всё в одной платформе</h2>
    <p class="sub">Приложение на телефоне монтажника. Сайт для руководителя. Одна база данных.</p>
    <div class="grid">
      <div class="card"><div class="icon">📦</div><h3>Склад под контролем</h3><p>Каждая единица на учёте: поступление, выдача, возврат, демонтаж. Каждый коллектив видит только свой склад.</p></div>
      <div class="card"><div class="icon">👷</div><h3>Бригады на виду</h3><p>Кто, где, когда и что делал — всегда видно. Работа через приложение, контроль — на сайте.</p></div>
      <div class="card"><div class="icon">💰</div><h3>Оплата по факту</h3><p>За метры и стыки, а не за часы. Споры исчезают. Тарифы и зарплаты — прозрачно.</p></div>
      <div class="card"><div class="icon">🔒</div><h3>Данные нельзя изменить</h3><p>SHA-256 цепочка событий, аудит, шифрование. Юридически значимые журналы.</p></div>
      <div class="card"><div class="icon">📊</div><h3>Отчёты за кнопку</h3><p>Вся картина по складу и деньгам. Экспорт, дашборд руководителя.</p></div>
      <div class="card"><div class="icon">🌐</div><h3>Работает везде</h3><p>Телефон, планшет, компьютер — Mac, Windows, Android, iOS. Всё синхронизировано.</p></div>
    </div>
    <a class="btn deck-link" href="/presentation">▶ Смотреть презентацию</a>
  </div>
</section>

<section class="section" id="pricing">
  <div class="container">
    <h2>Тарифы</h2>
    <p class="sub">14 дней бесплатного триала для каждой организации</p>
    <div class="tiers">
      <div class="tier">
        <div class="name">Бригада</div>
        <div class="amount">3 000 ₽<span style="font-size:1rem;color:var(--muted)">/мес</span></div>
        <div class="per">до 10 монтажников</div>
        <ul class="features">
          <li>Склад своей бригады</li>
          <li>Наряды и заявки</li>
          <li>Топливо и пробег</li>
          <li>Сайт руководителя</li>
        </ul>
        <a class="btn" href="#" id="subscribeBtn2">Оформить</a>
      </div>
      <div class="tier featured">
        <div class="name">Предприятие</div>
        <div class="amount">9 000 ₽<span style="font-size:1rem;color:var(--muted)">/мес</span></div>
        <div class="per">сводный контроль всех бригад</div>
        <ul class="features">
          <li>Всё из тарифа «Бригада»</li>
          <li>Неограниченно бригад</li>
          <li>Полная изоляция коллективов</li>
          <li>Приоритетная поддержка</li>
        </ul>
        <a class="btn" href="#" id="subscribeBtn3">Оформить</a>
      </div>
    </div>
  </div>
</section>

<section class="section" id="partner">
  <div class="container">
    <h2>Партнёрская программа</h2>
    <p class="sub">Приводите клиентов — получайте процент пожизненно</p>
    <div class="partner">
      <div class="big">20% <span style="font-size:1.2rem;color:var(--muted)">с каждого платежа — пожизненно</span></div>
      <p>У каждого партнёра своя реферальная ссылка. По ней новые клиенты получают скидку 30% на первый месяц, а партнёр — 20% с их подписки. Навсегда.</p>
      <p style="margin-top:8px;color:var(--gold)">Ссылка: <span id="partnerLink">—</span></p>
    </div>
  </div>
</section>

<section class="cta">
  <div class="container">
    <h2>Начните уже сегодня</h2>
    <p>14 дней бесплатно. Гарантия возврата 30 дней. Поддержка на каждом шаге.</p>
    <a class="btn large" href="#" id="ctaBtn">Оформить подписку</a>
  </div>
</section>

<footer>
  <div class="container">Монтажник Связи · 2026 · Платформа для монтажных бригад и подрядчиков</div>
</footer>

<script>
// Подхватываем реферальный код из URL (?ref=КОД) и показываем бейдж.
var ref = '{{REF}}';
if (ref) {
  var badge = document.getElementById('refBadge');
  badge.style.display = 'inline-block';
  document.getElementById('refCode').textContent = ref;
  document.getElementById('partnerLink').textContent = location.href.split('?')[0] + '?ref=' + ref;
  // Запоминаем код для перехода на регистрацию/оплату.
  localStorage.setItem('ref_code', ref);
}

// Кнопки «Оформить» — ведут на регистрацию с реферальным кодом.
function goSubscribe() {
  var r = localStorage.getItem('ref_code') || '';
  // Регистрация и подписка — общий аккаунт. С сайта открываем форму входа.
  window.location.href = 'http://192.168.0.10:8080/#/register?ref=' + encodeURIComponent(r);
}
document.getElementById('subscribeBtn').addEventListener('click', goSubscribe);
document.getElementById('subscribeBtn2').addEventListener('click', goSubscribe);
document.getElementById('subscribeBtn3').addEventListener('click', goSubscribe);
document.getElementById('ctaBtn').addEventListener('click', goSubscribe);
</script>
</body>
</html>`