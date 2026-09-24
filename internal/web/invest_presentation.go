package web

// InvestHTML — полная презентация «Монтажник Связи» как длинная веб-страница
// («портянка»): весь контент инвест-дека на 15 разделов, прокрутка сверху вниз.
// Это то же содержание, что в deck.js / montazhnik-svyazi-invest.pptx.
const InvestHTML = `<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Монтажник Связи — инвестиционная презентация</title>
<style>
:root {
  --bg: #0d1117; --panel: #161b22; --panel2: #10304F; --border: #30363d;
  --ink: #e6edf3; --muted: #8b949e; --accent: #58a6ff; --green: #3fb950;
  --gold: #d29922; --red: #f85149; --line: #21262d;
}
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, "Segoe UI", Roboto, Arial, sans-serif; background: var(--bg); color: var(--ink); line-height: 1.6; }
.wrap { max-width: 1000px; margin: 0 auto; padding: 40px 24px 80px; }

/* Шапка-титул */
.cover { text-align: center; padding: 60px 0 50px; border-bottom: 1px solid var(--border); margin-bottom: 40px; }
.cover h1 { font-size: clamp(2rem, 6vw, 3.4rem); margin-bottom: 18px; }
.cover .sub { font-size: clamp(1.1rem, 2.6vw, 1.5rem); color: var(--accent); font-weight: 700; max-width: 800px; margin: 0 auto 12px; }
.cover .meta { color: var(--muted); font-size: 1rem; }
.cover .tags { color: var(--accent); font-size: .95rem; margin-top: 20px; font-weight: 600; }

/* Блоки ("слайды" портянки) */
.block { background: var(--panel); border: 1px solid var(--border); border-radius: 16px; padding: 28px 32px; margin: 22px 0; }
.block .kicker { font-size: .85rem; letter-spacing: 3px; text-transform: uppercase; color: var(--accent); margin-bottom: 6px; }
.block h2 { font-size: clamp(1.4rem, 3vw, 1.9rem); margin-bottom: 18px; }
.block .lead { color: var(--accent); font-weight: 700; margin-bottom: 16px; font-size: 1.02rem; }
.block .text { color: var(--muted); font-size: .97rem; }
.block .text b { color: var(--ink); }

/* Две колонки (Сегодня/Результат, Мастер/Бухгалтер, Партнёры/Данные) */
.cols { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; }
.cols .col h3 { color: var(--accent); margin-bottom: 10px; font-size: 1.05rem; }
.cols .col ul { list-style: none; }
.cols .col ul li { padding: 6px 0 6px 22px; position: relative; color: var(--muted); font-size: .95rem; border-bottom: 1px solid var(--border); }
.cols .col ul li::before { content: "–"; position: absolute; left: 0; color: var(--accent); }

/* Нумерованные шаги (Сканирование→Криптоподпись, Партн. шаги) */
.steps { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px,1fr)); gap: 16px; margin-top: 8px; }
.step { text-align: center; background: var(--panel2); border: 1px solid var(--border); border-radius: 12px; padding: 16px 12px; }
.step .num { display: inline-flex; align-items: center; justify-content: center; width: 40px; height: 40px; border-radius: 50%; background: var(--accent); color: var(--bg); font-weight: 800; font-size: 1.1rem; margin-bottom: 8px; }
.step .t { font-size: .98rem; font-weight: 700; margin-bottom: 4px; }
.step .d { font-size: .82rem; color: var(--muted); }

/* Карточки-метрики */
.stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(170px,1fr)); gap: 16px; }
.stat { background: var(--panel2); border: 1px solid var(--border); border-radius: 12px; padding: 18px; text-align: center; }
.stat .v { font-size: 1.9rem; font-weight: 800; color: var(--accent); }
.stat .l { color: var(--muted); font-size: .85rem; margin-top: 4px; }
.stat.gold .v { color: var(--gold); }

/* Карточки-иконки (Логистика, Интеграции) */
.icons { display: grid; grid-template-columns: repeat(auto-fit, minmax(170px,1fr)); gap: 16px; }
.icard { background: var(--panel2); border: 1px solid var(--border); border-radius: 12px; padding: 16px; }
.icard .em { font-size: 1.5rem; margin-bottom: 8px; }
.icard .t { font-weight: 700; font-size: .98rem; margin-bottom: 4px; }
.icard .d { color: var(--muted); font-size: .85rem; }

/* Таблицы (Журналы, Технологии) */
table { width: 100%; border-collapse: collapse; font-size: .92rem; margin-top: 8px; }
th, td { text-align: left; padding: 10px 12px; border-bottom: 1px solid var(--border); }
th { background: var(--accent); color: var(--bg); font-weight: 700; }
td { color: var(--muted); }
td b { color: var(--ink); }
tr:nth-child(even) td { background: var(--panel2); }

/* Таймлайн */
.timeline { position: relative; margin: 20px 0 8px; height: 90px; border-top: 2px solid var(--accent); }
.timeline .point { position: absolute; top: -9px; }
.timeline .point .dot { width: 16px; height: 16px; border-radius: 50%; background: var(--accent); margin: 0 auto; }
.timeline .point .when { font-weight: 700; font-size: .85rem; margin-bottom: 6px; text-align: center; }
.timeline .point .what { color: var(--muted); font-size: .82rem; text-align: center; width: 120px; margin-left: -52px; }

/* Акцент-строка */
.accent-line { text-align: center; color: var(--accent); font-weight: 700; font-size: 1.02rem; margin-top: 18px; }

/* Финал */
.final { text-align: center; padding: 40px; border: 1px solid var(--accent); border-radius: 16px; margin-top: 30px; }
.final h2 { color: var(--accent); margin-bottom: 12px; font-size: 1.8rem; }
.final p { color: var(--muted); margin-bottom: 20px; }

@media (max-width: 700px) { .cols { grid-template-columns: 1fr; } }
</style>
</head>
<body>
<div class="wrap">

  <!-- Титул -->
  <div class="cover">
    <h1>МОНТАЖНИК СВЯЗИ</h1>
    <div class="sub">Единая цифровая среда для монтажных бригад: учёт работ, транспорта, зарплаты и партнёрская программа — в одном контуре</div>
    <div class="meta">Инвестиционное предложение · 2026</div>
    <div class="tags">Сканирование · Авторасчёт · SHA-256 · Офлайн-first · Интеграция с 1С · Партнёрам до 50%</div>
  </div>

  <!-- 2. Проблема -->
  <div class="block">
    <div class="kicker">Проблема</div>
    <h2>Почему полевая бригада связи тонет в бумаге и Excel</h2>
    <div class="cols">
      <div class="col"><h3>Сегодня</h3><ul>
        <li>Ручной ввод данных в Excel и 1С — опечатки и дубли</li>
        <li>Акты «на слово», фото без геопривязки, нет истории</li>
        <li>Непрозрачный расчёт зарплаты и споры с бригадой</li>
        <li>Нет контроля транспорта, маршрутов и ГСМ</li>
        <li>Привлечение клиентов только через дорогую рекламу</li>
      </ul></div>
      <div class="col"><h3>Результат</h3><ul>
        <li>Ошибки и задержки выплат</li>
        <li>Нет юридически значимых документов</li>
        <li>Расхождения склада до 15–20%</li>
        <li>Простои и срывы SLA</li>
        <li>Рост LTV ограничен без сети партнёров</li>
      </ul></div>
    </div>
  </div>

  <!-- 3. Решение: принцип работы -->
  <div class="block">
    <div class="kicker">Решение</div>
    <h2>Принцип работы: от сканирования до криптоподписи</h2>
    <div class="steps">
      <div class="step"><div class="num">1</div><div class="t">Сканирование</div><div class="d">QR/SKU материала, фотоотчёт с геотегом</div></div>
      <div class="step"><div class="num">2</div><div class="t">Ввод объёма</div><div class="d">100 м кабеля — единственное действие мастера</div></div>
      <div class="step"><div class="num">3</div><div class="t">Авторасчёт</div><div class="d">Время UTC, цена, НДС, сумма сдельной оплаты</div></div>
      <div class="step"><div class="num">4</div><div class="t">Криптоподпись</div><div class="d">SHA-256 хеш — данные неизменяемы</div></div>
    </div>
    <div class="accent-line">Офлайн-first: все данные сохраняются локально (SQLite) и синхронизируются при появлении связи — через 4G или Wi-Fi машины</div>
  </div>

  <!-- 4. Сквозной контур -->
  <div class="block">
    <div class="kicker">Архитектура</div>
    <h2>Сквозной контур: от наряда до ведомости в 1С</h2>
    <div class="steps">
      <div class="step"><div class="num">1</div><div class="t">Наряд-задание</div><div class="d">Автоматически приходит мастеру: адрес, работы, нормативы, расценки</div></div>
      <div class="step"><div class="num">2</div><div class="t">Фиксация работ</div><div class="d">Скан QR, фото, объём — данные с геотегом и UTC-временем</div></div>
      <div class="step"><div class="num">3</div><div class="t">Автоначисление</div><div class="d">Сумма = норматив × ставка × коэффициент. Видна мастеру сразу</div></div>
      <div class="step"><div class="num">4</div><div class="t">Выгрузка в 1С</div><div class="d">Документы «Начисление зарплаты» и «Акт выполненных работ»</div></div>
    </div>
    <div class="accent-line">Мастер делает минимум действий — система фиксирует, считает, подписывает и передаёт в бухгалтерию</div>
  </div>

  <!-- 5. Эффект -->
  <div class="block">
    <div class="kicker">Эффект для бизнеса</div>
    <h2>Что даёт автоматизация</h2>
    <div class="stats">
      <div class="stat"><div class="v">−80%</div><div class="l">ручного ввода данных</div></div>
      <div class="stat"><div class="v">−90%</div><div class="l">ошибок при оформлении</div></div>
      <div class="stat"><div class="v">70–80%</div><div class="l">экономии времени бухгалтерии</div></div>
      <div class="stat gold"><div class="v">+50%</div><div class="l">партнёрам от подписок</div></div>
    </div>
    <div class="accent-line">Прозрачность для мастера, контроль для диспетчера, готовые документы для бухгалтера</div>
  </div>

  <!-- 6. Логистика -->
  <div class="block">
    <div class="kicker">Логистика</div>
    <h2>Карты, транспорт и связь на трассе</h2>
    <div class="icons">
      <div class="icard"><div class="em">🗺️</div><div class="t">Карта заявок</div><div class="d">Диспетчер видит все наряды, статусы и геопозицию бригад</div></div>
      <div class="icard"><div class="em">🛣️</div><div class="t">Маршруты</div><div class="d">Оптимальный путь с учётом пробок и габаритов, офлайн-навигация</div></div>
      <div class="icard"><div class="em">⛽</div><div class="t">Учёт транспорта</div><div class="d">Пробег, ГСМ, фото-осмотр авто, автоматические путевые листы</div></div>
      <div class="icard"><div class="em">📶</div><div class="t">Wi-Fi машины</div><div class="d">Роутер с 2–3 SIM и усилителями — синхронизация без 4G</div></div>
    </div>
    <div class="accent-line">Связь между бригадами — через приложение, без радиосвязи: чат, статусы, фото и координаты</div>
  </div>

  <!-- 7. Неизменяемые журналы -->
  <div class="block">
    <div class="kicker">Контроль</div>
    <h2>Неизменяемые журналы — основа аудита</h2>
    <table>
      <thead><tr><th>Журнал</th><th>Назначение</th><th>Ключевое поле</th></tr></thead>
      <tbody>
        <tr><td>Операций</td><td>Все движения ТМЦ и работ</td><td><b>SHA-256 хеш + UTC</b></td></tr>
        <tr><td>Статусов нарядов</td><td>Жизненный цикл задач</td><td>Привязка к наряду</td></tr>
        <tr><td>Инвентаризаций</td><td>Сверка остатков</td><td>План vs Факт</td></tr>
        <tr><td>Связи и синхронизации</td><td>Доставка данных</td><td>Канал, сигнал, статус</td></tr>
        <tr><td>Криптоцелостности</td><td>Проверка неизменности</td><td>Исходный и текущий хеш</td></tr>
      </tbody>
    </table>
    <div class="accent-line">Любая правка задним числом ломает цепочку хешей — это юридически значимая доказательная база</div>
  </div>

  <!-- 8. Партнёрская программа -->
  <div class="block">
    <div class="kicker">Рост выручки</div>
    <h2>Партнёрская программа: до 50% от подписок</h2>
    <div class="steps">
      <div class="step"><div class="num">1</div><div class="t">Регистрация</div><div class="d">ФИО, ИНН, статус — проверка через API ФНС</div></div>
      <div class="step"><div class="num">2</div><div class="t">Ссылка</div><div class="d">Уникальный промокод для привлечения</div></div>
      <div class="step"><div class="num">3</div><div class="t">Оплата</div><div class="d">Подписчик платит за приложение</div></div>
      <div class="step"><div class="num">4</div><div class="t">Автовыплата</div><div class="d">50% на счёт партнёра автоматически</div></div>
    </div>
    <div class="accent-line">Масштабирование без роста ФОТ: партнёры сами продают — реестр выплат, акты и налоги формируются автоматически</div>
  </div>

  <!-- 9. Учёт зарплаты -->
  <div class="block">
    <div class="kicker">Мотивация</div>
    <h2>Индивидуальный заработок — прозрачно</h2>
    <div class="cols">
      <div class="col"><h3>Личный кабинет мастера</h3><ul>
        <li>Автовход по токенам — без пароля</li>
        <li>Своя страница с начислениями</li>
        <li>Формула: норматив × ставка × коэффициент</li>
        <li>Форма защищена после подтверждения</li>
        <li>История за неделю/месяц, PDF</li>
      </ul></div>
      <div class="col"><h3>Для бухгалтера</h3><ul>
        <li>Готовые данные из форм мастеров</li>
        <li>Перенос в 1С или Excel — ведомость</li>
        <li>Авторасчёт НДФЛ 13/15% и взносов</li>
        <li>Выгрузка реестра АВР с подписью</li>
        <li>Провал из ведомости до документа</li>
      </ul></div>
    </div>
  </div>

  <!-- 10. Интеграции -->
  <div class="block">
    <div class="kicker">Интеграции</div>
    <h2>Закрытый контур, который уже умеет работать вместе</h2>
    <div class="icons">
      <div class="icard"><div class="em">🏢</div><div class="t">1С (ЗУП / УТ)</div><div class="d">Зарплата, путевые листы, агентские вознаграждения</div></div>
      <div class="icard"><div class="em">🏦</div><div class="t">Банк</div><div class="d">Реестры выплат, прямые платёжные поручения через API</div></div>
      <div class="icard"><div class="em">📄</div><div class="t">API ФНС</div><div class="d">Проверка статуса партнёра (ИП / самозанятый)</div></div>
      <div class="icard"><div class="em">📍</div><div class="t">Карты OSM</div><div class="d">Векторные тайлы, маршруты, офлайн-навигация</div></div>
      <div class="icard"><div class="em">📡</div><div class="t">Телематика</div><div class="d">OBD-II, датчики топлива, спектроанализаторы</div></div>
    </div>
    <div class="accent-line">Все коммуникации передаются по TLS, каждая операция подписывается — данные не покидают закрытый контур</div>
  </div>

  <!-- 11. Масштабируемость -->
  <div class="block">
    <div class="kicker">Масштабируемость</div>
    <h2>Тысячи ведомостей — без утяжеления устройства</h2>
    <div class="stats">
      <div class="stat"><div class="v">10–30 МБ</div><div class="l">данных на устройстве мастера</div></div>
      <div class="stat gold"><div class="v">~295 МБ</div><div class="l">текст на 620 000 записей</div></div>
      <div class="stat"><div class="v">S3 / MinIO</div><div class="l">фото хранятся отдельно от БД</div></div>
      <div class="stat"><div class="v">100+</div><div class="l">мастеров без деградации</div></div>
    </div>
    <div class="accent-line">Локальная БД (SQLite) — офлайн-буфер. Сервер (PostgreSQL) — источник истины. Фото — только по требованию</div>
  </div>

  <!-- 12. Дорожная карта -->
  <div class="block">
    <div class="kicker">План</div>
    <h2>Дорожная карта внедрения</h2>
    <div style="position:relative;margin:40px 0 10px;border-top:2px solid var(--accent);height:100px;">
      <div class="timeline-item" style="position:absolute;left:0;top:-9px;"><div class="dot" style="width:16px;height:16px;border-radius:50%;background:var(--accent);"></div><div style="font-weight:700;margin:8px 0 2px;">Месяц 1</div><div style="color:var(--muted);font-size:.85rem;">Пилот на 1 бригаде</div></div>
      <div class="timeline-item" style="position:absolute;left:25%;top:-9px;"><div class="dot" style="width:16px;height:16px;border-radius:50%;background:var(--accent);"></div><div style="font-weight:700;margin:8px 0 2px;">Месяц 2</div><div style="color:var(--muted);font-size:.85rem;">Интеграция с 1С</div></div>
      <div class="timeline-item" style="position:absolute;left:50%;top:-9px;"><div class="dot" style="width:16px;height:16px;border-radius:50%;background:var(--accent);"></div><div style="font-weight:700;margin:8px 0 2px;">Месяц 3</div><div style="color:var(--muted);font-size:.85rem;">Масштабирование</div></div>
      <div class="timeline-item" style="position:absolute;left:75%;top:-9px;"><div class="dot" style="width:16px;height:16px;border-radius:50%;background:var(--accent);"></div><div style="font-weight:700;margin:8px 0 2px;">Месяц 6</div><div style="color:var(--muted);font-size:.85rem;">Партнёрская сеть</div></div>
      <div class="timeline-item" style="position:absolute;right:0;top:-9px;"><div class="dot" style="width:16px;height:16px;border-radius:50%;background:var(--accent);"></div><div style="font-weight:700;margin:8px 0 2px;">2027</div><div style="color:var(--muted);font-size:.85rem;">ИИ и цифровой рубль</div></div>
    </div>
  </div>

  <!-- 13. Технологии -->
  <div class="block">
    <div class="kicker">Технологии</div>
    <h2>Стек, на котором это построено</h2>
    <table>
      <thead><tr><th>Слой</th><th>Технология</th><th>Задача</th></tr></thead>
      <tbody>
        <tr><td>Мобильное приложение</td><td>Flutter (Dart)</td><td>Android 15+, офлайн, камера, BLE, GPS</td></tr>
        <tr><td>Бэкенд</td><td>Go (Gin + pgx)</td><td>Высокая скорость и конкурентность</td></tr>
        <tr><td>Базы данных</td><td>PostgreSQL + SQLite</td><td>Сервер и локальный кэш</td></tr>
        <tr><td>Файлы</td><td>S3 / MinIO</td><td>Фото, SOR-файлы, акты</td></tr>
        <tr><td>ИИ / LLM</td><td>Локальная GGUF-модель</td><td>База знаний, OCR, подсказки</td></tr>
        <tr><td>Криптография</td><td>SHA-256, ECDSA</td><td>Неизменяемость и подпись</td></tr>
      </tbody>
    </table>
    <div class="accent-line">Один бинарный файл работает на Windows 11 и переносится на Android 15+</div>
  </div>

  <!-- 14. Право -->
  <div class="block">
    <div class="kicker">Право</div>
    <h2>Юридически чистая модель выплат и данных</h2>
    <div class="cols">
      <div class="col"><h3>Партнёрские выплаты</h3><ul>
        <li>Агентский договор / публичная оферта</li>
        <li>Выплаты — расходы компании (счёт 44/26), не личный доход</li>
        <li>НДФЛ 13/15% — налоговый агент у физлиц без статуса</li>
        <li>ИП и самозанятые платят налоги сами</li>
        <li>Назначение платежа и реестр — для спокойного банка</li>
      </ul></div>
      <div class="col"><h3>Данные и аудит</h3><ul>
        <li>152-ФЗ: шифрование, роли, сроки хранения</li>
        <li>63-ФЗ: электронная подпись для значимых документов</li>
        <li>SHA-256 и UTC-время — неизменяемая история</li>
        <li>Ролевая модель: мастер видит только свой профиль</li>
        <li>Готовность к цифровому рублю для автовыплат</li>
      </ul></div>
    </div>
  </div>

  <!-- 15. Closing -->
  <div class="final">
    <h2>Готовы к пилоту</h2>
    <p>Демо бригаде из 3–5 человек за 2 недели. Покажем на реальных заявках: наряд → сканирование → автоначисление → ведомость.</p>
    <div class="stats">
      <div class="stat"><div class="l"><b style="color:var(--accent)">Что нужно</b><br>Доступ к справочникам: номенклатура, ставки, тарифы</div></div>
      <div class="stat"><div class="l"><b style="color:var(--accent)">Ответственный</b><br>Куратор со стороны компании на пилот</div></div>
      <div class="stat"><div class="l"><b style="color:var(--accent)">Срок</b><br>Пилот: 2 недели · Интеграция 1С: +2 недели</div></div>
    </div>
    <p style="margin-top:24px;color:var(--accent);font-weight:700;">МОНТАЖНИК СВЯЗИ · 2026</p>
  </div>

</div>
</body>
</html>`