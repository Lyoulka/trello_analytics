# Proposal

## Why

Backend-метрики бесполезны тимлиду без интерфейса: Dashboard, задачи, история, отчёты и настройки. Нужен Vue 3 UI, который только отображает данные API без дублирования формул.

## What Changes

- Экраны: login, Dashboard, графики, таблица карточек, история карточки, отчёты периода, настройки workflow/sync, управление пользователями (Admin).
- Единый HTTP-клиент, состояния loading/empty/error, предупреждения о качестве данных.
- Библиотека графиков: Chart.js + vue-chartjs (закрытие OQ-005 для v1).
- Копирование текстового отчёта в буфер.

## Capabilities

### New Capabilities

- `analytics-ui`: пользовательский интерфейс аналитики, отчётов, настроек и статусов на Vue 3.

### Modified Capabilities

- (нет)

## Impact

**Затронутые репозитории:**

- `trello_analytics_front` — страницы, router, api client, компоненты.
- `trello_analytics_open_spec` — artifacts.
- `trello_analytics_back` — **вне scope** (контракты уже в `auth-and-api`).

**Вне scope:** LLM-тексты отчётов, мульти-борд, production Nginx.
