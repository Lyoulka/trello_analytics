# Design

## Context

См. proposal.md. API контракты — `auth-and-api`. Frontend scaffold уже есть (Vue 3 + Vite + Router). ADR-004: формулы только с backend.

Текущие зависимости `trello_analytics_front`: `vue`, `vue-router` (+ Vite/TS toolchain). UI-kit и CSS-framework пока не подключены; стили — минимальный `src/styles.css`.

Визуальный референс: [shadcn/vue components](https://www.shadcn-vue.com/docs/components) — компоновка, палитра, типографика, отступы, карточки, таблицы, навигация и паттерны состояний. Референс задаёт направление, а не копирование чужого продукта.

## Goals / Non-Goals

**Goals:**

- Полноценный UI v1: auth, dashboard, charts, cards, history, reports, settings, sync status.
- Единый API client и UX состояний (loading / empty / error).
- Единый визуальный язык для Dashboard, карточек Trello, отчётов и настроек — аналитический инструмент для Team Lead.
- Адаптивность: desktop-first, usable на планшете.

**Non-Goals:**

- Дублирование формул метрик на клиенте.
- LLM-генерация текстов.
- Nuxt.
- Смешение нескольких UI-библиотек (Element Plus + Vuetify + Naive и т.п.).
- Изменение backend-контрактов ради UI.
- Подстановка выдуманных метрик в рабочие вызовы `/api/v1`.

## Decisions

1. **Структура страниц:** `src/pages/` — Login, Dashboard, Cards, CardHistory, Reports, Settings, Users (Admin).
2. **State:** Composition API + лёгкий store при необходимости (Pinia, если уже не выбран — добавить Pinia для session/filters).
3. **Charts (OQ-005):** Chart.js + `vue-chartjs`. Стили графиков (цвета серий, оси, легенда, пустые состояния) согласовать с токенами дизайн-системы UI, не вводя вторую chart-библиотеку.
4. **Фильтры периода:** общий composable (`from`/`to`/preset) согласованно для dashboard/charts/tables.
5. **Копирование отчёта:** клиент собирает текстовый preview из API `reports/preview` в clipboard.
6. **Роли в UI:** скрывать недоступные действия; решение всегда на API.
7. **Data quality banners:** показывать warnings из API (incomplete history, workflow missing, sync stale).
8. **UI kit (закрытие open question):** **shadcn-vue + Tailwind CSS** как единственный компонентный подход.
   - Совместим со стеком Vue 3 + TypeScript + Vite; установка по [Vite guide](https://www.shadcn-vue.com/docs/installation/vite).
   - Компоненты копируются в репозиторий (`src/components/ui/…`), не как «чёрный ящик» npm-UI-kit.
   - Не подключать параллельно другие полные UI frameworks.
   - Chart.js остаётся отдельной зависимостью только для графиков (не конкурирующий UI kit).
9. **Единый дизайн для Team Lead analytics:**
   - **Компоновка:** app shell с боковой/верхней навигацией (референс Sidebar / Navigation Menu), основная область — KPI-ряд + графики + таблицы; плотная, но читаемая сетка; один визуальный ритм на всех экранах.
   - **Палитра:** нейтральная база (shadcn Neutral / аналог), семантические цвета success / warning / danger / muted для статусов sync и data quality; без декоративных «маркетинговых» градиентов.
   - **Типографика:** чёткая иерархия заголовков/метрик/подписей (референс Typography); крупные числа KPI, компактный secondary text для дат и пояснений.
   - **Отступы:** единая шкала spacing (Tailwind); карточки и секции с предсказуемыми gaps; таблицам — достаточный row padding без «воздушных» лендинговых полей.
   - **Карточки:** Card для KPI, секций отчёта, блоков настроек; не превращать каждый виджет в тяжёлый shadow-блок.
   - **Таблицы:** Data Table / Table + Pagination / filters (Select, Input, Date Picker) для Cards и списков.
   - **Графики:** внутри Card; loading → Skeleton; empty → Empty; error → Alert.
   - **Навигация:** стабильный набор пунктов Dashboard / Cards / Reports / Settings (+ Users для Admin); активный маршрут явно выделен.
10. **UX-состояния (обязательны на каждом data-экране):**
    - **Loading:** Skeleton / Spinner — не пустой белый экран.
    - **Empty:** Empty + короткое объяснение (нет данных за период / нет карточек по фильтру).
    - **Error:** Alert (и при необходимости retry); без утечки внутренних деталей API.
    - **Partial / warning:** banner из API data-quality (не маскировать нулями).
11. **Демонстрационные данные:** допустимы только в явно помеченных макетах / Story-подобных страницах / fixture-превью с лейблом вроде «Demo / mock». Рабочие экраны, ходящие в `/api/v1`, MUST NOT подставлять выдуманные метрики. `null` / missing ≠ `0`.
12. **Перед реализацией экранов (task 2.0):** зафиксировать токены (цвета, радиусы, типографика), список используемых shadcn-vue primitives и карту экранов → компоненты; сверить с текущим `package.json`, чтобы не плодить лишние зависимости.

### Рекомендуемый набор shadcn-vue (референс → применение)

| Область | Референс-компоненты | Экраны |
|---------|---------------------|--------|
| Shell / nav | Sidebar, Navigation Menu, Breadcrumb, Separator | все защищённые |
| KPI / секции | Card, Badge, Tooltip | Dashboard, Reports |
| Формы / фильтры | Button, Input, Select, Date Picker, Calendar, Label, Switch, Field | Login, фильтры, Settings |
| Таблицы | Table / Data Table, Pagination, Checkbox | Cards, Users |
| Feedback | Alert, Skeleton, Spinner, Empty, Toast/Sonner, Progress | все data-экраны |
| Оверлеи | Dialog, Sheet, Dropdown Menu | подтверждения, детали |
| Графики | Chart (визуальный паттерн) + наша связка Chart.js | Dashboard, Reports |

## Risks / Trade-offs

- [Расхождение UI прав и API] → Mitigation: опираться на `me.role` + обработку 403.
- [Большие таблицы] → Mitigation: серверная пагинация, не грузить всю историю на клиент.
- [Tailwind + shadcn увеличивают объём tooling] → Mitigation: один подход вместо ad-hoc CSS и нескольких UI kits; компоненты только по мере нужды (`shadcn-vue add …`).
- [Соблазн demo-метрик в UI до готовности API] → Mitigation: явный mock-лейбл; иначе только реальные ответы / честный empty/error.

## Open Questions

- (закрыто) UI kit → Decision 8: shadcn-vue + Tailwind CSS.
- Плотность таблицы Cards по умолчанию (comfortable vs compact) — уточнить при первом UI-pass с тимлидом; до этого — comfortable desktop.
