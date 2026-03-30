# Task 7. HTTP sync, uploads и health endpoints в `api/`

## Статус
Выполнена.

## Цель
Подготовить endpoint-first HTTP-контракты и каркас обработчиков для sync, chunk uploads/downloads и health-check.

## Описание
Синхронизация между устройствами и работа с бинарными данными входят в исходную идею проекта, но заметно сложнее обычного CRUD. Поэтому их нужно держать отдельной задачей с явно описанными DTO для курсоров, pending-операций, upload session и resume-сценариев.

## Последовательность шагов
1. Создать пакеты `api/sync_pull_v1_post`, `api/sync_push_v1_post`, `api/uploads_v1_post`, `api/uploads_by_id_chunks_v1_post`, `api/uploads_by_id_v1_get`, `api/health_v1_get`.
2. В каждом пакете подготовить `handler.go`, `handler_test.go` и `mocks/`.
3. Описать DTO для sync cursors, pending changes, conflict response, upload session, chunk state и download response.
4. Зафиксировать правила resume после обрыва для upload и download.
5. Определить mapping ошибок для конфликта sync, незавершенной загрузки и некорректного порядка чанков.
6. Добавить unit-тесты на типовые и ошибочные сценарии.

## Критерии приемки
- [x] Для sync, uploads и health есть отдельные endpoint-first пакеты в `api/`.
- [x] HTTP-контракты отражают chunk upload/download и resume.
- [x] Sync DTO покрывают курсоры, pending changes и конфликты.
- [x] Обработаны ошибки незавершенной загрузки и некорректного порядка чанков.
- [x] На каждый endpoint есть базовые unit-тесты.
