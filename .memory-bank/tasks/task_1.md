# Task 1. Каркас репозитория и MVP-границы

## Статус
Выполнено.

## Цель
Привести структуру репозитория к целевому layout из `.memory-bank/structure-recommendations.md` и явно зафиксировать границы MVP: первая поставка включает только `server + cli`.

## Описание
Текущий backlog должен вести к реализации проекта из `.memory-bank/project-idea.md` и `.memory-bank/plan.md`, а не размывать scope. На этом шаге нужно выровнять layout репозитория, создать обязательные каталоги для MVP и оставить только архитектурные заглушки для `desktop` и `web` без продуктовой реализации. Задача не включает разработку бизнес-логики.

## Последовательность шагов
1. Сверить текущую структуру репозитория с `.memory-bank/structure-recommendations.md` и составить список расхождений.
2. Привести layout к целевой схеме: `cmd/server`, `cmd/client/cli`, `internal/models`, `internal/services`, `internal/rpc`, `api`, `rpc/proto/v1`, `pkg/apiclient`, `pkg/clientcore`, `pkg/cache`, `migrations`, `tests`.
3. Создать архитектурные заглушки для post-MVP направлений: `cmd/client/desktop`, `cmd/client/web`, `tests/integration`, `tests/e2e`.
4. Убедиться, что структура каталогов не создает ложного впечатления, будто `desktop` и `web` входят в MVP.
5. Обновить базовые комментарии и README-фрагменты, где нужно явно указать: MVP = `server + cli`, `desktop/web` = backlog.

## Критерии приемки
- [x] Структура каталогов соответствует `.memory-bank/structure-recommendations.md` в части MVP.
- [x] Есть отдельные entrypoint-каталоги для `cmd/server` и `cmd/client/cli`.
- [x] Есть только структурные заглушки для `cmd/client/desktop` и `cmd/client/web` без продуктовой логики.
- [x] Доменные, transport- и прикладные слои физически разделены по пакетам.
- [x] В документации явно зафиксировано, что MVP включает только `server + cli`.
