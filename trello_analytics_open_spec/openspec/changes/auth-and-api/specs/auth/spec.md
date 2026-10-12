# Spec Delta

## Purpose

Обеспечивает локальную авторизацию пользователей приложения, роли Admin/Team Lead/Member и серверную проверку прав без публичной саморегистрации.

## ADDED Requirements

### Requirement: Вход и текущий пользователь
Система MUST предоставлять login, logout и получение текущего пользователя; пароли MUST храниться только в виде безопасного hash.

#### Scenario: Успешный login
- **WHEN** пользователь передаёт верные email и пароль
- **THEN** система MUST установить аутентифицированную сессию и позволить `GET /api/v1/auth/me`

#### Scenario: Неверный пароль
- **WHEN** пароль неверен
- **THEN** система MUST отказать без раскрытия, существует ли email, и без утечки hash

### Requirement: Bootstrap Admin и создание пользователей
Первый Admin MUST создаваться bootstrap-механизмом при деплое/старте, не через публичную регистрацию; остальные пользователи MUST создаваться Admin.

#### Scenario: Нет публичной регистрации
- **WHEN** клиент пытается зарегистрироваться через публичный signup endpoint
- **THEN** такого endpoint MUST NOT быть в v1

#### Scenario: Admin создаёт пользователя
- **WHEN** Admin создаёт пользователя с ролью
- **THEN** запись MUST появиться в app_users и пользователь сможет войти

### Requirement: Ролевая проверка на сервере
Каждый защищённый endpoint MUST проверять роль на backend; frontend ограничения MUST NOT считаться достаточными.

#### Scenario: Member не запускает sync
- **WHEN** пользователь с ролью Member вызывает start sync
- **THEN** API MUST вернуть отказ в доступе

#### Scenario: Секреты Trello
- **WHEN** клиент читает settings
- **THEN** Trello token/key MUST NOT возвращаться в открытом виде
