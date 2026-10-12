# Spec Delta

## Purpose

Задаёт расчёт командных метрик доставки и качества на backend из сохранённой истории Trello по правилам workflow без подмены отсутствующих событий.

## ADDED Requirements

### Requirement: Completed и Throughput по уникальным карточкам
Система MUST считать количество уникальных карточек, завершённых в периоде по правилу done; Throughput MUST отражать завершённые задачи, а не число actions.

#### Scenario: Перемещение между done-lists
- **WHEN** карточка перемещается между двумя done-списками без возврата в work
- **THEN** она MUST NOT учитываться дважды в completed/Throughput периода

#### Scenario: Фильтры периода
- **WHEN** задан период (и опционально member/label/type)
- **THEN** метрики MUST учитывать только применимые фильтры

### Requirement: Lead Time и Cycle Time
Система MUST рассчитывать Lead Time от настроенного start rule до done и Cycle Time как календарную разницу от начала выполнения до done; календарное время MUST NOT выдаваться за active working time.

#### Scenario: Медиана и среднее
- **WHEN** в периоде есть завершённые карточки с полными границами
- **THEN** API/сервис MUST уметь предоставить среднее и медиану (и при реализации — перцентили)

#### Scenario: Нет события start
- **WHEN** у карточки отсутствует достоверное событие начала
- **THEN** карточка MUST исключаться из Lead/Cycle или помечаться как incomplete, без выдуманной даты

### Requirement: Возвраты багов и время исправления
Система MUST определять reopen бага по bug classification + возврат из done в work; считать количество, процент по когорте, fix time до следующего done; циклы хранить раздельно.

#### Scenario: Done-to-done без work
- **WHEN** карточка двигается между done-статусами без входа в work
- **THEN** это MUST NOT считаться reopen

#### Scenario: Неоднозначный reopen
- **WHEN** событие возврата нельзя достоверно определить
- **THEN** карточка MUST попасть в список на проверку, а не в silent расчёт

### Requirement: Stale, WIP и workload
Система MUST рассчитывать незавершённые/зависшие задачи по порогу и распределению по участникам с явными правилами multi-assignee; stale MUST формулироваться как отсутствие активности в Trello, не как доказательство бездействия человека.

#### Scenario: Порог stale
- **WHEN** карточка без движения дольше настроенного порога
- **THEN** она MUST включаться в stale-выборку

### Requirement: Документация формул
Формулы, исключения и граничные случаи MUST быть описаны в `docs/analytics-metrics.md`.

#### Scenario: Документ существует
- **WHEN** change завершён
- **THEN** `docs/analytics-metrics.md` MUST содержать формулы ключевых метрик и правила incomplete history
