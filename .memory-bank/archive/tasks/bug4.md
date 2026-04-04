# Название
Исправить удаление записей

# Статус: DONE

# Описание
Сейчас пробовал удалить запись и столкнулся с ошибкой grpc. Лог ошибки также прикладываю, надо будет это поправить.
```
{"level":"info","method":"/gophkeeper.v1.DataService/DeleteRecord","duration":3.263458,"error":"rpc error: code = NotFound desc = record not found","time":"2026-04-04T09:01:47+03:00","message":"grpc unary request"}
```

# Исследование
- Запись была удалена из серверного хранилища (soft delete), но осталась в локальном кэше клиента
- При повторной попытке удаления клиент шлёт DeleteRecord gRPC-запрос
- `records.Service.GetRecord` (строка 61) фильтрует soft-deleted записи и возвращает `ErrRecordNotFound`
- `DataService.DeleteRecord` пробрасывал эту ошибку как gRPC NotFound, хотя операция delete уже выполнена

# Причина
Delete — идемпотентная операция. Если запись не существует или уже удалена — это успех, а не ошибка. Старый код возвращал `FailedPrecondition` для уже удалённых записей и `NotFound` для несуществующих.

# Анализ смежных путей
Проверен путь UpdateRecord:
- `UpdateRecord` также получает `ErrRecordNotFound` для soft-deleted записей, но это **корректное поведение** — update не идемпотентен, нельзя молча обновить удалённую запись
- Код `existing.IsDeleted()` на строке 151 — мёртвый код (GetRecord уже отфильтровал soft-deleted), но не влияет на корректность
- Sync push (`sync.Service.pushUpdate`) использует `recordRepo.GetRecord` напрямую и корректно обрабатывает soft-deleted записи (восстановление + конфликты)

HTTP delete handler имел ту же проблему — исправлен.

# Исправление

## gRPC: `internal/rpc/data_service.go`
- `ErrRecordNotFound` при GetRecord -> возвращаем успех (запись не существует или уже soft-deleted)
- `record.IsDeleted()` -> возвращаем успех (запись уже удалена)
- Проверка владельца (`UserID != userID`) сохранена для существующих неудалённых записей

## HTTP: `internal/api/records_by_id_v1_delete/handler.go`
- Та же логика: `ErrRecordNotFound` и `record.IsDeleted()` возвращают 200 OK

## Обновлённые тесты

### `internal/rpc/data_service_test.go`
- `TestDataService_DeleteRecord_NotFound` -> `TestDataService_DeleteRecord_NotFound_Idempotent` (теперь ожидает успех)
- `TestDataService_DeleteRecord_AlreadyDeleted` -> `TestDataService_DeleteRecord_AlreadyDeleted_Idempotent` (теперь ожидает успех)

### `internal/api/records_by_id_v1_delete/handler_test.go`
- "record not found" -> "record not found - idempotent" (теперь ожидает 200)
- "already deleted" -> "already deleted - idempotent" (теперь ожидает 200)

# Результат
- Изменено файлов: 2 (`internal/rpc/data_service.go`, `internal/api/records_by_id_v1_delete/handler.go`)
- Обновлено тестов: 2 (`internal/rpc/data_service_test.go`, `internal/api/records_by_id_v1_delete/handler_test.go`)
- Все 187 тестов (177 rpc + 10 HTTP delete) проходят
- Сборка проекта (`go build ./...`) успешна
