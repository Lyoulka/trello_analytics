# Spec Delta — Bug reopen QA review

## MODIFIED Requirements

### Requirement: Возвраты багов и время исправления
Система MUST определять reopen бага как переход карточки с bug-label из stage ∈ `bug_reopen_from_stages` в stage ∉ `done_stages`; MUST считать числитель и знаменатель как **числа переходов** (не уникальные карточки); знаменатель MUST быть числом переходов багов в `done_stages` за период; переход из reopen-from в done MUST NOT считаться reopen; done→done MUST NOT увеличивать знаменатель.

#### Scenario: Reopen из «на проверку» не в Done
- **WHEN** баг уходит из stage ∈ `bug_reopen_from_stages` в non-done в периоде
- **THEN** это MUST увеличить числитель reopen на 1 (за каждый такой переход)

#### Scenario: Уход на проверку → Done
- **WHEN** баг уходит из `bug_reopen_from_stages` в done
- **THEN** это MUST NOT считаться reopen; MUST увеличить знаменатель (вход в done), если переход в периоде

#### Scenario: Процент по переходам
- **WHEN** в периоде есть N reopen-переходов и M done-enter переходов багов
- **THEN** reopen % MUST быть `(N/M)×100` при M>0; при M=0 percent MUST NOT выдаваться как достоверный 0%

#### Scenario: Нет bug_reopen_from_stages
- **WHEN** `bug_reopen_from_stages` пуст
- **THEN** bug reopen MUST быть unavailable с предупреждением конфигурации
