# Название
Убрать жесткую связь хендлеров и сервисов

# Статус
DONE

# Описание
Если посмотреть в сервис sync (internal/services/sync), там можно увидеть зависимость от хендлера internal/api/sync_push_v1_post. Такого не должно быть. Сервисы не должны зависеть от хендлеров, так же как и хендлеры в явном медиа не должны зависеть от сервисов. Хендлеры должны работать только с интерфейсами и моделями, точно так же как и сервисы должны работать только с интерфейсами и моделями. В данном случае сервис sync должен был использовать какие-то модели, которые должны находиться в internal/models.
В текущей же задаче требуется пройтись по всем сервисам в internal/services, найти такие же подобные места и убрать жесткую зависимость от хендлеров в пользу использования моделей, интерфейсов и тому подобное.

# Промежуточный прогресс

## Анализ
- Прошёлся по всем 11 сервисам в `internal/services/`
- По импортам выявлено: только **sync** сервис имеет зависимость от `internal/api/`
- Остальные 10 сервисов (`records`, `auth`, `users`, `keys`, `crypto`, `data`, `passwords`, `uploads`, `validation`, `docs`) — чистые, без зависимостей от хендлеров

## Найденные проблемы в sync
1. `services/sync/service.go` импортировал `internal/api/sync_push_v1_post` и `internal/api/records_common`
2. Метод `Push` принимал `[]sync_push_v1_post.PendingChange` (DTO из хендлера)
3. Внутри сервиса была функция `dtoPayloadToDomain()` для конвертации API DTO → доменные модели
4. Интерфейс `SyncPusher` был определён в пакете хендлера, а не в сервисном слое
5. В `server.go` был `syncUseCaseAdapter` с конвертацией `rpc.PendingChange` → `sync_push_v1_post.PendingChange` и `recordToDTO()`

## Выполненные изменения

### 1. `internal/models/sync.go`
- Добавлена модель `models.PendingChange` с полями `Record *Record`, `Deleted bool`, `BaseRevision int64`

### 2. `internal/services/sync/service.go`
- Убраны импорты `internal/api/records_common` и `internal/api/sync_push_v1_post`
- Сигнатура `Push` и все внутренние методы переписаны на `models.PendingChange`
- Удалены функции `dtoPayloadToDomain()` и `strVal()` — конвертация DTO→domain перенесена в хендлер
- Добавлена проверка `change.Record == nil` в `pushChange`

### 3. `internal/api/sync_push_v1_post/handler.go`
- Интерфейс `SyncPusher.Push` теперь принимает `[]models.PendingChange`
- Добавлены функции конвертации `toDomainChanges()`, `dtoToDomainRecord()`, `dtoPayloadToDomain()`, `strVal()` — конвертация DTO→domain теперь на стороне хендлера

### 4. `internal/rpc/sync_service.go`
- Удалён тип `rpc.PendingChange` (дублировал `models.PendingChange`)
- Интерфейс `SyncUseCase.Push` теперь принимает `[]models.PendingChange`

### 5. `internal/app/server.go`
- Интерфейс `SyncService.Push` теперь принимает `[]models.PendingChange`
- Удалён `syncUseCaseAdapter` — интерфейсы стали идентичны, передаём `deps.SyncService` напрямую в `rpc.NewSyncService`
- Удалена функция `recordToDTO()` и убраны импорты `records_common`

### 6. Обновлены тесты
- `internal/services/sync/service_test.go` — переписан на использование `models.PendingChange` с `*models.Record`
- `internal/services/sync/service_integration_test.go` — аналогично
- `internal/app/server_test.go` — удалены тесты `syncUseCaseAdapter` и `recordToDTO` (код удалён)
- `internal/api/sync_push_v1_post/handler_test.go` — `mockSyncPusher` переписан на `[]models.PendingChange`
- `internal/rpc/sync_service_test.go` — заменён `PendingChange` → `models.PendingChange`
- `tests/e2e/metadata_roundtrip_test.go` — заменён `rpc.PendingChange` → `models.PendingChange`

### 7. Перегенерированы моки
- `internal/app/mocks/sync_service_mock.go`
- `internal/api/sync_push_v1_post/mocks/sync_pusher_mock.go`

# Итоговая статистика
- Сервисов проверено: 11
- Сервисов с зависимостью от хендлеров найдено: 1 (sync)
- Файлов изменено: ~14
- Моков перегенерировано: 2
- Тестов пройдено: 938 (internal) + 43 (e2e) = 981
- Строк кода удалено (неточных данных, грубо): ~200 (adapter + recordToDTO + dtoPayloadToDomain + тесты)
