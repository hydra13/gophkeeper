# gophkeeper

Менеджер паролей GophKeeper — серверное хранилище секретов с CLI-, TUI-, desktop- и web-клиентами.

## MVP scope

**Входит в первую поставку:**
- Сервер (`cmd/server`) — HTTP REST + gRPC API, TLS-only
- CLI-клиент (`cmd/client/cli`) — командный интерфейс
- TUI-клиент (`cmd/client/tui`) — интерактивный терминальный интерфейс на `tview`
- Web-клиент (`cmd/client/web`) — браузерный интерфейс на `React + TypeScript + Vite + Ant Design`
- Типы записей: логин/пароль, текст, бинарные данные, банковские карты
- Server-side шифрование (AES-256-GCM, envelope encryption с data keys)
- Ротация ключей и фоновое перешифрование данных
- Двунаправленная синхронизация между клиентами (push/pull)
- Chunk-загрузка бинарных файлов с поддержкой resume
- PostgreSQL для хранения, S3-compatible blob-хранилище для бинарных данных

**Не входит в MVP (вынесено в backlog):**
- Web-клиент
- OTP/TOTP
- Импорт/экспорт данных

## Архитектура

```
cmd/
  server/           — точка входа сервера
  client/cli/       — точка входа CLI-клиента
  client/tui/       — точка входа TUI-клиента
  client/desktop/   — desktop-клиент на Wails + React
  client/common/    — общий bootstrap клиента
internal/
  api/              — HTTP-обработчики (по одному пакету на endpoint)
  rpc/              — gRPC-сервисы и protobuf-контракты
  services/         — бизнес-логика (auth, records, sync, uploads, crypto, keys)
  repositories/     — слой хранения (PostgreSQL, in-memory, file)
  models/           — доменные сущности
  config/           — загрузка и валидация конфигурации
  middlewares/      — HTTP/gRPC middleware (auth, TLS, rate limit, logging)
  jobs/reencrypt/   — фоновый job перешифрования данных
  storage/          — blob-хранилище (local, S3-compatible)
  migrations/       — applying goose-миграций
pkg/
  clientcore/       — use-case слой клиента (офлайн, pending-ops, sync)
  clientui/         — общие helper-функции для CLI/TUI
  apiclient/        — транспортный слой клиента (gRPC; HTTP — заглушка, post-MVP)
  cache/            — клиентское локальное хранилище состояния
  buildinfo/        — версия и дата сборки
migrations/         — SQL-миграции (goose, встроены в бинарник)
configs/            — примеры конфигурации и dev-сертификаты
```

## Быстрый старт

### Требования

- Go 1.25+
- PostgreSQL 15+
- protoc + protoc-gen-go + protoc-gen-go-grpc (для генерации из proto)
- golangci-lint, goimports

### 1. Запуск PostgreSQL и MinIO

```bash
make dev-up
```

Команда поднимает:

- `PostgreSQL` на `localhost:5432`
- `MinIO` на `localhost:9000`
- web-консоль `MinIO` на `http://localhost:9001`
- bucket `gophkeeper-dev`, который создаётся автоматически до завершения `make dev-up`

Остановить локальную инфраструктуру:

```bash
make dev-down
```

### 2. Конфигурация

Сервер по умолчанию читает `configs/config.dev.json` (уже настроен для локальной разработки с dev-сертификатами). Если нужно изменить параметры, отредактируйте этот файл или переопределите через переменные окружения:

| Поле                    | Env                       | Описание                            |
| ----------------------- | ------------------------- | ----------------------------------- |
| `server.address`        | `GK_SERVER_ADDRESS`       | Адрес HTTP-сервера (`:8080`)        |
| `server.grpc_address`   | `GK_GRPC_ADDRESS`         | Адрес gRPC-сервера (`:9090`)        |
| `server.tls_cert_file`  | `GK_TLS_CERT_FILE`        | Путь к TLS-сертификату сервера      |
| `server.tls_key_file`   | `GK_TLS_KEY_FILE`         | Путь к TLS-ключу сервера            |
| `database.dsn`          | `GK_DATABASE_DSN`         | DSN PostgreSQL                      |
| `auth.jwt_secret`       | `GK_JWT_SECRET`           | Секрет для подписи JWT              |
| `auth.token_duration`   | `GK_TOKEN_DURATION`       | Срок жизни access token (ns)        |
| `crypto.master_key`     | `GK_MASTER_KEY`           | 32-байтный мастер-ключ (base64 или raw, raw-строка должна быть длиной ровно 32 байта) |
| `blob.provider`         | `GK_BLOB_PROVIDER`        | Провайдер blob-хранилища (`local` или `s3`) |
| `blob.path`             | `GK_BLOB_PATH`            | Директория для local blob-хранилища |
| `blob.endpoint`         | `GK_BLOB_ENDPOINT`        | S3 endpoint (`http://localhost:9000`) |
| `blob.bucket`           | `GK_BLOB_BUCKET`          | Имя bucket-а                        |
| `blob.access_key`       | `GK_BLOB_ACCESS_KEY`      | S3 access key                       |
| `blob.secret_key`       | `GK_BLOB_SECRET_KEY`      | S3 secret key                       |
| `blob.region`           | `GK_BLOB_REGION`          | S3 region                           |
| `upload.max_file_size`  | —                         | Макс. размер файла (байт)           |
| `upload.max_chunk_size` | —                         | Макс. размер чанка (байт)           |

`configs/config.dev.json` по умолчанию настроен на локальный `MinIO`:

- `blob.provider = "s3"`
- `blob.endpoint = "http://localhost:9000"`
- `blob.bucket = "gophkeeper-dev"`
- `blob.access_key = "minioadmin"`
- `blob.secret_key = "minioadmin"`
- `blob.region = "us-east-1"`

CLI-, TUI- и desktop-клиенты используют переменные окружения для адреса и сертификата:

| Переменная          | По умолчанию               | Описание                              |
| ------------------- | -------------------------- | ------------------------------------- |
| `GK_GRPC_ADDRESS`   | `localhost:9090`           | Адрес gRPC-сервера                    |
| `GK_TLS_CERT_FILE`  | `configs/certs/dev.crt`    | Путь к CA-сертификату для TLS         |

Для разработки в `configs/certs/` лежат самоподписанные сертификаты. Клиенты автоматически подхватят `configs/certs/dev.crt` при запуске из корня проекта. Сервер, CLI и TUI должны запускаться из корня репозитория, чтобы пути к сертификатам разрешались корректно.

### 3. Сборка и запуск сервера

```bash
make build-server
./bin/server
```

Сервер использует fail-fast модель: при отсутствии обязательной зависимости (DSN, подключение к БД, миграции, blob storage) процесс завершается с ненулевым кодом и диагностическим сообщением. Миграции применяются автоматически при старте (goose, встроены в бинарник). Если БД недоступна, миграции не удалось применить, `MinIO`/S3 недоступен, bucket не существует или credentials некорректны — сервер не запустится.

### 4. Сборка и запуск CLI

CLI подключается к серверу по TLS. По умолчанию используется `configs/certs/dev.crt` как CA-сертификат — запускайте из корня проекта.

```bash
make build-client-cli
./bin/client register user@example.com
./bin/client login user@example.com
./bin/client add login "Мой сайт"
./bin/client list
./bin/client sync
```

Если сертификат расположен в другом месте, укажите путь через переменную окружения:

```bash
GK_TLS_CERT_FILE=/path/to/ca.crt ./bin/client register user@example.com
```

### 5. Сборка и запуск TUI

TUI использует тот же `ClientCore`, тот же локальный cache и те же TLS-параметры, что и CLI. При старте доступны `Login`, `Register`, `Exit`; после входа доступны `list/get/add/update/delete/sync/logout`.

```bash
make build-client-tui
./bin/client-tui
```

Для быстрого запуска без сборки бинарника:

```bash
go run ./cmd/client/tui
```

Горячие клавиши на основном экране:

- `l` — обновить список
- `g` — открыть запись или сохранить binary на диск
- `a` — добавить запись
- `u` — обновить выбранную запись
- `d` — удалить выбранную запись
- `s` — синхронизация
- `o` — logout
- `q` — выход

### 6. Сборка и запуск desktop

Desktop-клиент использует тот же `ClientCore`, тот же локальный cache и тот же TLS bootstrap, что и CLI/TUI. На старте доступны только `Login` и `Register`; после успешного `register` показывается modal `registered successfully`, затем автоматически выполняется login.

Для локальной сборки нужен установленный Node.js 20+ и Wails CLI.

```bash
cd cmd/client/desktop/frontend
npm install
npm run build
cd ../../..
env CGO_LDFLAGS="-framework UniformTypeIdentifiers $CGO_LDFLAGS" \
  go run -tags production ./cmd/client/desktop
```

Или через `make`:

```bash
make build-client-desktop
./bin/client-desktop
```

### 7. Запуск web-клиента

Web-клиент работает как отдельное Vite-приложение в [`cmd/client/web`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/web) и по умолчанию ходит в сервер через proxy на `https://localhost:8080`.

Для локальной разработки:

```bash
make dev-client-web
```

После запуска открой адрес `http://localhost:34116`.

Production-сборка web-клиента:

```bash
make build-client-web
```

Важно: `Wails` требует специальные build tags. Поэтому direct-сборка и direct-запуск должны идти с `-tags production`, например:

```bash
env CGO_LDFLAGS="-framework UniformTypeIdentifiers $CGO_LDFLAGS" \
  go build -tags production -o bin/client-desktop ./cmd/client/desktop
./bin/client-desktop
```

## CLI-команды

```
gophkeeper-cli register [email]       Регистрация нового пользователя
gophkeeper-cli login [email]          Вход (получение токенов)
gophkeeper-cli logout                 Выход (отзыв токенов)
gophkeeper-cli list [type]            Список записей (login|text|binary|card)
gophkeeper-cli get name <name> [path] Получить запись по имени
gophkeeper-cli get id <id> [path]     Получить запись по ID
gophkeeper-cli add <type> <name>      Добавить запись
gophkeeper-cli update name <name> <new-name> [data] Обновить запись по имени
gophkeeper-cli update id <id> <new-name> [data]     Обновить запись по ID
gophkeeper-cli delete name <name>                   Удалить запись по имени
gophkeeper-cli delete id <id>                       Удалить запись по ID
gophkeeper-cli sync                   Синхронизация с сервером
gophkeeper-cli version                Версия CLI
```

### Metadata

Любой записи можно задать произвольную текстовую metadata (дополнительное описание, заметки):

```bash
# Создание записи с metadata
gophkeeper-cli add text "Мой сайт" --metadata "Рабочий аккаунт"

# Обновление metadata без изменения payload
gophkeeper-cli update id 5 "Мой сайт" --metadata "Новая заметка"

# Очистка metadata
gophkeeper-cli update id 5 "Мой сайт" --metadata ""
```

## HTTP API

Все запросы — только через TLS. Публичные endpoints: `/api/v1/auth/*`, `/api/v1/health`. Остальные требуют JWT в заголовке `Authorization: Bearer <token>`.

| Метод | Путь                              | Описание                |
| ----- | --------------------------------- | ----------------------- |
| POST  | `/api/v1/auth/register`           | Регистрация             |
| POST  | `/api/v1/auth/login`              | Вход                    |
| POST  | `/api/v1/auth/refresh`            | Обновление токена       |
| POST  | `/api/v1/auth/logout`             | Выход                   |
| GET   | `/api/v1/health`                  | Проверка здоровья       |
| POST  | `/api/v1/records`                 | Создание записи         |
| GET   | `/api/v1/records`                 | Список записей          |
| GET   | `/api/v1/records/{id}`            | Получение записи        |
| PUT   | `/api/v1/records/{id}`            | Обновление записи       |
| DELETE| `/api/v1/records/{id}`            | Удаление записи         |
| POST  | `/api/v1/sync/push`               | Отправка изменений      |
| POST  | `/api/v1/sync/pull`               | Получение изменений     |
| POST  | `/api/v1/uploads`                 | Создание upload-сессии  |
| GET   | `/api/v1/uploads/{id}`            | Статус загрузки         |
| POST  | `/api/v1/uploads/{id}/chunks`     | Загрузка чанка          |
| GET   | `/api/v1/uploads/{id}/chunks/{i}` | Скачивание чанка        |

## Smoke-сценарий

Проверка работоспособности после запуска:

```bash
# 1. Проверка health
curl -k https://localhost:8080/api/v1/health

# 2. Регистрация
curl -k -X POST https://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"secret123"}'

# 3. Вход
curl -k -X POST https://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"secret123"}'

# 4. Создание записи (используйте access_token из шага 3)
curl -k -X POST https://localhost:8080/api/v1/records \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"type":"login","name":"test","payload":{"login":"user","password":"pass"}}'

# 5. Список записей
curl -k https://localhost:8080/api/v1/records \
  -H "Authorization: Bearer <access_token>"
```

Проверка, что локальный blob backend поднят:

```bash
curl http://localhost:9000/minio/health/live
```

## Команды Makefile

| Команда            | Описание                                      |
| ------------------ | --------------------------------------------- |
| `make fmt`         | Форматирование кода через goimports           |
| `make lint`        | Статический анализ (golangci-lint)            |
| `make test`        | Запуск тестов с race detector и покрытием     |
| `make cover`       | Генерация HTML-отчёта покрытия                |
| `make cover-check` | Проверка порога покрытия (>= 70%)             |
| `make proto`       | Генерация Go-кода из protobuf                 |
| `make build`       | Сборка server + client                        |
| `make build-server`| Сборка только сервера                         |
| `make build-client`| Сборка обоих клиентов: CLI + TUI              |
| `make build-client-cli`| Сборка только CLI-клиента                |
| `make build-client-tui`| Сборка только TUI-клиента                |
| `make clean`       | Удаление артефактов сборки                    |
| `make dev-up`      | Поднять PostgreSQL + MinIO через Docker Compose |
| `make dev-down`    | Остановить локальную инфраструктуру           |
| `make dev-reset`   | Остановить и удалить локальные volumes        |
| `make test-storage-integration` | Проверить S3 backend против локального MinIO |

## Тестирование

```bash
# Юнит-тесты (без PostgreSQL)
make test

# Проверка реального S3/MinIO roundtrip (требуется make dev-up)
make test-storage-integration

# Проверка порога покрытия >= 70%
make cover-check

# С интеграционными тестами (требуется запущенный PostgreSQL)
GK_TEST_DATABASE_DSN="postgres://postgres:postgres@localhost:5432/gophkeeper?sslmode=disable" make test
```

Определение метрики покрытия и список исключений см. в `docs/coverage-policy.md`.

## Безопасность

- Все соединения — TLS-only (HTTP и gRPC)
- Пароли хранятся как bcrypt-хеши
- Данные шифруются server-side (AES-256-GCM)
- Envelope encryption: мастер-ключ шифрует data keys, data keys шифруют данные
- JWT access/refresh token с ротацией refresh
- Soft delete записей
- Монотонный рост ревизий для обнаружения конфликтов синхронизации
