# Рекомендуемая структура проекта GophKeeper

## Контекст
Структура ниже приведена в соответствие с текущим `plan.md`:
- MVP = `server + cli`;
- сервер поддерживает два равноправных transport слоя: `gRPC` и `HTTP REST`;
- данные шифруются на сервере по схеме `master-key + периодические data keys`;
- бинарные вложения сразу поддерживают chunk upload/download и resume;
- клиентский кеш допускается в открытом виде;
- `desktop` и `web` закладываются архитектурно, но не являются обязательной частью первой поставки.

## Рекомендуемая структура директорий

```text
gophkeeper/
├── rpc/                                   # Контракты gRPC transport слоя
│   └── proto/
│       └── v1/
│           ├── auth.proto                 # gRPC auth API
│           ├── records.proto              # CRUD и metadata
│           ├── sync.proto                 # Синхронизация
│           └── uploads.proto              # Chunk upload/download
│
├── api/                                   # HTTP endpoint-first структура
│   ├── auth_register_v1_post/
│   │   ├── handler.go
│   │   ├── handler_test.go
│   │   └── mocks/
│   ├── auth_login_v1_post/
│   │   ├── handler.go
│   │   ├── handler_test.go
│   │   └── mocks/
│   ├── auth_refresh_v1_post/
│   │   ├── handler.go
│   │   ├── handler_test.go
│   │   └── mocks/
│   ├── auth_logout_v1_post/
│   │   ├── handler.go
│   │   ├── handler_test.go
│   │   └── mocks/
│   ├── records_v1_get/
│   │   ├── handler.go
│   │   ├── handler_test.go
│   │   └── mocks/
│   ├── records_v1_post/
│   │   ├── handler.go
│   │   ├── handler_test.go
│   │   └── mocks/
│   ├── records_by_id_v1_get/
│   │   ├── handler.go
│   │   ├── handler_test.go
│   │   └── mocks/
│   ├── records_by_id_v1_put/
│   │   ├── handler.go
│   │   ├── handler_test.go
│   │   └── mocks/
│   ├── records_by_id_v1_delete/
│   │   ├── handler.go
│   │   ├── handler_test.go
│   │   └── mocks/
│   ├── sync_pull_v1_post/
│   │   ├── handler.go
│   │   ├── handler_test.go
│   │   └── mocks/
│   ├── sync_push_v1_post/
│   │   ├── handler.go
│   │   ├── handler_test.go
│   │   └── mocks/
│   ├── uploads_v1_post/
│   │   ├── handler.go
│   │   ├── handler_test.go
│   │   └── mocks/
│   ├── uploads_by_id_chunks_v1_post/
│   │   ├── handler.go
│   │   ├── handler_test.go
│   │   └── mocks/
│   ├── uploads_by_id_v1_get/
│   │   ├── handler.go
│   │   ├── handler_test.go
│   │   └── mocks/
│   └── health_v1_get/
│       ├── handler.go
│       ├── handler_test.go
│       └── mocks/
│
├── cmd/
│   ├── server/
│   │   └── main.go                        # Запуск HTTP + gRPC сервера
│   └── client/
│       ├── cli/
│       │   └── main.go                    # Основной клиент MVP
│       ├── desktop/                       # Каркас под Wails, post-MVP
│       │   └── .gitkeep
│       └── web/                           # Каркас web-клиента, post-MVP
│           └── .gitkeep
│
├── configs/
│   ├── config.example.json
│   ├── config.example.yaml
│   └── tls/
│       ├── dev-cert.pem
│       └── dev-key.pem
│
├── internal/
│   ├── app/
│   │   ├── server.go                      # Bootstrap transport слоёв
│   │   ├── cli.go                         # Bootstrap CLI приложения
│   │   └── shutdown.go                    # Graceful shutdown
│   │
│   ├── config/
│   │   ├── config.go
│   │   ├── loader.go
│   │   └── validate.go
│   │
│   ├── domain/
│   │   ├── user.go                        # Пользователь
│   │   ├── record.go                      # Базовая запись
│   │   ├── record_types.go                # login/text/binary/card
│   │   ├── metadata.go                    # Произвольная metadata
│   │   ├── session.go                     # device-aware sessions
│   │   ├── sync.go                        # Ревизии и конфликты
│   │   ├── upload.go                      # Upload session/chunks
│   │   ├── key_version.go                 # active/deprecated/retired
│   │   └── errors.go
│   │
│   ├── grpc/
│   │   ├── server.go
│   │   ├── auth_service.go
│   │   ├── records_service.go
│   │   ├── sync_service.go
│   │   └── uploads_service.go
│   │
│   ├── middlewares/
│   │   ├── auth.go                        # HTTP auth middleware
│   │   ├── logger.go                      # Логирование запросов/ответов
│   │   ├── ratelimit.go
│   │   └── tls.go                         # Проверка transport-конфига
│   │
│   ├── interceptors/
│   │   ├── auth.go                        # gRPC auth interceptor
│   │   ├── logger.go
│   │   └── ratelimit.go
│   │
│   ├── repositories/
│   │   ├── users.go
│   │   ├── records.go
│   │   ├── sessions.go
│   │   ├── sync_state.go
│   │   ├── uploads.go
│   │   ├── key_versions.go
│   │   ├── postgres/
│   │   │   ├── users.go
│   │   │   ├── records.go
│   │   │   ├── sessions.go
│   │   │   ├── uploads.go
│   │   │   └── key_versions.go
│   │   └── fileblob/
│   │       └── storage.go                 # Файловое хранение binary payload
│   │
│   ├── services/
│   │   ├── auth/
│   │   │   ├── service.go                 # Email/password auth
│   │   │   └── tokens.go                  # Access/refresh tokens
│   │   ├── records/
│   │   │   └── service.go
│   │   ├── sync/
│   │   │   └── service.go
│   │   ├── uploads/
│   │   │   └── service.go                 # Chunk upload/download + resume
│   │   ├── crypto/
│   │   │   ├── encryptor.go               # Шифрование данных
│   │   │   ├── key_manager.go             # Master-key + period keys
│   │   │   └── reencrypt.go               # Batch re-encryption
│   │   └── validation/
│   │       └── service.go
│   │
│   └── jobs/
│       └── reencrypt/
│           └── job.go                     # Фоновая ротация и перешифрование
│
├── pkg/
│   ├── clientcore/                        # Shared client core
│   │   ├── auth.go
│   │   ├── records.go
│   │   ├── sync.go
│   │   ├── cache.go
│   │   └── uploads.go
│   ├── apiclient/
│   │   ├── grpc/
│   │   │   └── client.go
│   │   └── http/
│   │       └── client.go
│   ├── cache/
│   │   ├── metadata.go
│   │   ├── records.go
│   │   ├── pending.go
│   │   └── uploads.go                     # Незавершённые upload/download
│   └── buildinfo/
│       └── info.go                        # Версия и дата сборки
│
├── migrations/
│   ├── 20260321000001_init_schema.up.sql
│   ├── 20260321000001_init_schema.down.sql
│   ├── 20260321000002_sessions.up.sql
│   ├── 20260321000002_sessions.down.sql
│   ├── 20260321000003_uploads.up.sql
│   ├── 20260321000003_uploads.down.sql
│   ├── 20260321000004_key_versions.up.sql
│   └── 20260321000004_key_versions.down.sql
│
├── tests/
│   ├── integration/
│   │   ├── grpc/
│   │   ├── http/
│   │   └── tls/
│   └── e2e/
│       ├── cli/
│       ├── sync/
│       ├── uploads/
│       └── reencrypt/
│
├── .github/
│   └── workflows/
│       ├── test.yml
│       └── lint.yml
│
├── .gitignore
├── .golangci.yml
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

## Что важно учесть в структуре

### 1. Домен отделён от transport слоя
`internal/domain` и `internal/services` не должны зависеть от HTTP или gRPC деталей.
Оба transport слоя используют один и тот же use-case слой.

### 2. HTTP и gRPC равноправны
Нельзя сводить HTTP к gateway-обёртке, потому что по утверждённому плану это отдельный полноценный API.
Из-за этого нужны отдельные endpoint-пакеты в `api/`, отдельный `internal/grpc/` и отдельные integration tests для обоих transport слоёв.

### 3. HTTP-слой проектируется endpoint-first
Каждая HTTP-ручка живёт в отдельном пакете по шаблону `api/<path через нижнее подчеркивание>_<версия>_<http-метод>`.
Внутри пакета должны лежать как минимум `handler.go`, `handler_test.go` и каталог `mocks/` под зависимости конкретной ручки.

### 4. Chunk upload закладывается сразу
Бинарные вложения требуют отдельной модели `upload session`, состояния чанков и локального восстановления после обрыва.
Поэтому выделены `uploads` в домене, сервисах, transport-контрактах, кеше и тестах.

### 5. Ротация ключей является частью core-архитектуры
`key_versions`, `jobs/reencrypt` и crypto-сервисы нельзя оставлять как технический долг.
Это обязательная часть MVP.

### 6. Shared client core ориентирован на CLI, но не одноразовый
`pkg/clientcore` должен обслуживать текущий CLI и оставаться точкой переиспользования для будущего desktop-клиента.
`web` проектируется отдельно позже и не должен ломать эту границу.

## Рекомендации по корневым файлам

### Makefile
Минимально нужны команды:
- `build-server`
- `build-cli`
- `test`
- `test-integration`
- `lint`
- `proto`
- `migrate-up`
- `migrate-down`

### .gitignore
Дополнительно имеет смысл игнорировать:
```gitignore
bin/
dist/
coverage.out
*.db
*.sqlite
configs/config.local.json
configs/tls/*.pem
.env
```

### README.md
README должен отражать текущий scope:
- `server + cli` как MVP;
- два transport слоя: gRPC и HTTP REST;
- TLS only;
- device-aware sessions;
- chunk upload/resume для binary data;
- отсутствие import/export в MVP.

## Что изменить относительно старой рекомендации

### Удалить или не делать частью MVP
1. Ставку только на gRPC как основной transport.
2. Клиентское обязательное шифрование локального кеша.
3. Отдельный корневой `cmd/cli/`, если клиентская иерархия уже зафиксирована как `cmd/client/{cli,desktop,web}`.
4. Упрощённую модель binary storage без upload sessions.

### Добавить обязательно
1. `rpc/proto/` для gRPC контрактов.
2. Endpoint-first пакеты в `api/` для HTTP-ручек.
3. `internal/interceptors/` для gRPC middleware.
4. `internal/services/uploads/` и `internal/domain/upload.go`.
5. `internal/services/crypto/key_manager.go`.
6. `internal/jobs/reencrypt/`.
7. `pkg/clientcore/` и `pkg/cache/uploads.go`.
7. `tests/integration/http`, `tests/integration/grpc`, `tests/e2e/uploads`, `tests/e2e/reencrypt`.

## Почему эта структура лучше подходит текущему плану
1. Она отражает реальный MVP, а не более ранние предположения о single-transport архитектуре.
2. Она не прячет критичные решения вроде chunk upload и key rotation в “потом”.
3. Она оставляет расширяемость для `desktop` и `web`, но не раздувает первую поставку лишними обязательствами.
