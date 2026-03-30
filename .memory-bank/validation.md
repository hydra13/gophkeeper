# Валидация соответствия проекта исходной идее

Дата повторной проверки: 2026-03-30

## Что перепроверено

Повторная валидация выполнена по тем же источникам, которые перечислены в задаче:

- `.memory-bank/project-idea.md`
- `README.md`
- `docs/release-readiness.md`
- актуальный код репозитория
- воспроизводимые проверки из репозитория

Отдельно перепроверены четыре области из remediation-цикла:

1. работа CLI с `metadata`;
2. согласованность TLS quick start;
3. fail-fast bootstrap сервера;
4. выполнение порога покрытия 70%.

## Выполненные проверки

- `go test ./cmd/client/cli -run 'TestDefault' -count=1` — unit-тесты TLS defaults и сертификатного bootstrap
- `go test ./cmd/client/cli -run 'TestDefaultNewCoreNoCertError' -count=1` — fail-fast при отсутствии TLS-сертификата
- `go test ./cmd/client/cli -run 'Test.*Metadata' -count=1` — metadata-сценарии
- `go test ./cmd/server -run 'Test.*(WireDeps|Main|Fail)' -count=1` — серверный fail-fast
- `make cover-check`

Результаты:

- `TestDefaultServerAddr`: без env → `localhost:9090`, с `GK_GRPC_ADDRESS` → значение env.
- `TestDefaultTLSCertFile`: с `GK_TLS_CERT_FILE` → путь из env; без env, нет файла → `""`; без env, есть `configs/certs/dev.crt` → этот путь.
- `TestDefaultNewCoreNoCertError`: при отсутствии сертификата возвращается ошибка `TLS certificate is required...`.
- CLI metadata-тесты проходят.
- Серверные тесты по `wireDeps` и fail-fast-поведению проходят.
- `make cover-check` проходит и фиксирует `75.6%` по метрике, описанной в `docs/coverage-policy.md`.

## Итог

По состоянию на 2026-03-30 проект соответствует исходной идее в тех областях, которые были проблемными в предыдущей валидации. Критические расхождения из предыдущего отчёта устранены.

## Что подтверждено

### 1. Metadata в CLI подтверждена

Требование из ТЗ о произвольной текстовой метаинформации для любых записей теперь закрыто.

Факты:

- CLI явно поддерживает `--metadata` в `add` и `update`: `cmd/client/cli/main.go`
- Поддержаны сценарии установки, обновления и очистки metadata.
- Есть покрытие тестами для text/login/card/binary и metadata-only update: `cmd/client/cli/cli_test.go`
- Есть e2e roundtrip-проверки metadata: `tests/e2e/metadata_roundtrip_test.go`

Статус прежнего замечания: **устранено**.

### 2. TLS quick start и CLI defaults согласованы

Клиент и сервер больше не расходятся по умолчаниям. Defaults и сертификатный bootstrap CLI подтверждены unit-тестами.

Факты:

- CLI по умолчанию использует `localhost:9090`: `cmd/client/cli/main.go:159-164`
- CLI ищет `configs/certs/dev.crt` и требует TLS-сертификат для подключения: `cmd/client/cli/main.go:166-174`
- При отсутствии сертификата CLI завершает работу с явной ошибкой, а не уходит в insecure-режим через CLI bootstrap.
- README описывает тот же адрес gRPC и тот же dev-сертификат в quick start.
- Сервер поднимает HTTP и gRPC только с TLS: `internal/app/server.go`

Покрытие unit-тестами (`cmd/client/cli/cli_test.go`):

- `TestDefaultServerAddr`: проверяет default `localhost:9090` и override через `GK_GRPC_ADDRESS`.
- `TestDefaultTLSCertFile`: проверяет env override, fallback на `configs/certs/dev.crt` и возврат пустой строки при отсутствии сертификата.
- `TestDefaultNewCoreNoCertError`: подтверждает fail-fast при отсутствии TLS-сертификата.

Отдельный e2e TLS quick-start smoke (сервер + клиент + реальное TLS-подключение) не выполнялся.

Статус прежнего замечания: **устранено**.

### 3. Fail-fast bootstrap подтверждён

Сервер больше не стартует в режиме заглушек при проблемах с зависимостями.

Факты:

- Пустой `database.dsn` приводит к ошибке wiring: `cmd/server/main.go`
- Ошибка подключения к БД приводит к ошибке wiring: `cmd/server/main.go`
- Ошибка миграций приводит к ошибке wiring: `cmd/server/main.go`
- Эти сценарии покрыты тестами: `cmd/server/main_test.go`

Статус прежнего замечания: **устранено**.

### 4. Порог покрытия подтверждён

Требование по покрытию подтверждено в рамках зафиксированной проектной метрики.

Факты:

- `make cover-check` завершился успешно.
- Итог команды: `Coverage (excl. mocks, generated): 75.6%`.
- Правило расчёта метрики явно зафиксировано в `docs/coverage-policy.md` и используется в `Makefile`.

Статус прежнего замечания: **устранено в рамках принятой метрики проекта**.

## Остаточные ограничения / backlog

Повторная валидация не выявила новых критических расхождений относительно `.memory-bank/project-idea.md` в рамках remediation-областей. Оставшиеся ограничения относятся к уже явно вынесенному за пределы MVP или ранее зафиксированному backlog:

- HTTP transport client остаётся post-MVP заглушкой, основной клиентский транспорт — gRPC.
- Ротация ключей и re-encryption остаются эксплуатационными процедурами без отдельной пользовательской CLI-команды.
- Desktop/Web-клиенты, OTP/TOTP, TUI, импорт/экспорт данных остаются вне MVP.

## Согласованность документации

После этой повторной проверки:

- `README.md` соответствует фактическим CLI/TLS/defaults и fail-fast-поведению.
- `docs/release-readiness.md` должен отражать только подтверждённые в этой проверке утверждения.
- backlog и MVP scope не противоречат текущему состоянию кода.

## Вывод

Remediation-цикл по спорным местам закрыт: `metadata` в CLI, TLS quick start, fail-fast bootstrap и порог покрытия подтверждены повторной проверкой. Обновлённый `docs/release-readiness.md` можно использовать как консервативный источник правды для передачи проекта.
