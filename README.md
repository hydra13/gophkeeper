# gophkeeper

Менеджер паролей GophKeeper — серверное хранилище секретов с CLI-клиентом.

## MVP scope

**Входит в первую поставку:**
- Сервер (`cmd/server`) — HTTP REST + gRPC API, TLS-only
- CLI-клиент (`cmd/client/cli`) — интерактивный терминальный клиент
- Типы записей: логин/пароль, текст, бинарные данные, банковские карты
- Server-side шифрование (AES-256-GCM, envelope encryption с data keys)
- Ротация ключей и фоновое перешифрование данных
- Двунаправленная синхронизация между клиентами (push/pull)
- Chunk-загрузка бинарных файлов с поддержкой resume
- PostgreSQL для хранения, локальное файловое хранилище для blob

**Не входит в MVP (вынесено в backlog):**
- Desktop-клиент (GUI)
- Web-клиент
- OTP/TOTP
- TUI-интерфейс
- Импорт/экспорт данных

## Архитектура

```
cmd/
  server/           — точка входа сервера
  client/cli/       — точка входа CLI-клиента
internal/
  api/              — HTTP-обработчики (по одному пакету на endpoint)
  rpc/              — gRPC-сервисы и protobuf-контракты
  services/         — бизнес-логика (auth, records, sync, uploads, crypto, keys)
  repositories/     — слой хранения (PostgreSQL, in-memory, file)
  models/           — доменные сущности
  config/           — загрузка и валидация конфигурации
  middlewares/      — HTTP/gRPC middleware (auth, TLS, rate limit, logging)
  jobs/reencrypt/   — фоновый job перешифрования данных
  storage/          — локальное blob-хранилище
  migrations/       — applying goose-миграций
pkg/
  clientcore/       — use-case слой клиента (офлайн, pending-ops, sync)
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

### 1. Запуск PostgreSQL

```bash
docker run -d \
  --name gophkeeper-postgres \
  -p 5432:5432 \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=gophkeeper \
  -e PGDATA=/var/lib/postgresql/data/pgdata \
  -v ./.db:/var/lib/postgresql/data \
  postgres:15-alpine
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
| `crypto.master_key`     | `GK_MASTER_KEY`           | 32-байтный мастер-ключ (base64 или raw) |
| `blob.path`             | —                         | Директория для blob-хранилища       |
| `upload.max_file_size`  | —                         | Макс. размер файла (байт)           |
| `upload.max_chunk_size` | —                         | Макс. размер чанка (байт)           |

CLI-клиент использует переменные окружения для адреса и сертификата:

| Переменная          | По умолчанию               | Описание                              |
| ------------------- | -------------------------- | ------------------------------------- |
| `GK_GRPC_ADDRESS`   | `localhost:9090`           | Адрес gRPC-сервера                    |
| `GK_TLS_CERT_FILE`  | `configs/certs/dev.crt`    | Путь к CA-сертификату для TLS         |

Для разработки в `configs/certs/` лежат самоподписанные сертификаты. CLI автоматически подхватит `configs/certs/dev.crt` при запуске из корня проекта. Сервер и CLI должны запускаться из корня репозитория, чтобы пути к сертификатам разрешались корректно.

### 3. Сборка и запуск сервера

```bash
make build-server
./bin/server
```

Сервер использует fail-fast модель: при отсутствии обязательной зависимости (DSN, подключение к БД, миграции) процесс завершается с ненулевым кодом и диагностическим сообщением. Миграции применяются автоматически при старте (goose, встроены в бинарник). Если БД недоступна или миграции не удалось применить — сервер не запустится.

### 4. Сборка и запуск CLI

CLI подключается к серверу по TLS. По умолчанию используется `configs/certs/dev.crt` как CA-сертификат — запускайте из корня проекта.

```bash
make build-client
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

## CLI-команды

```
gophkeeper-cli register [email]       Регистрация нового пользователя
gophkeeper-cli login [email]          Вход (получение токенов)
gophkeeper-cli logout                 Выход (отзыв токенов)
gophkeeper-cli list [type]            Список записей (login|text|binary|card)
gophkeeper-cli get <id> [path]        Получить запись по ID
gophkeeper-cli add <type> <name>      Добавить запись
gophkeeper-cli update <id> <name>     Обновить запись
gophkeeper-cli delete <id>            Удалить запись
gophkeeper-cli sync                   Синхронизация с сервером
gophkeeper-cli version                Версия CLI
```

### Metadata

Любой записи можно задать произвольную текстовую metadata (дополнительное описание, заметки):

```bash
# Создание записи с metadata
gophkeeper-cli add text "Мой сайт" --metadata "Рабочий аккаунт"

# Обновление metadata без изменения payload
gophkeeper-cli update 5 "Мой сайт" --metadata "Новая заметка"

# Очистка metadata
gophkeeper-cli update 5 "Мой сайт" --metadata ""
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
| `make build-client`| Сборка только CLI-клиента                     |
| `make clean`       | Удаление артефактов сборки                    |

## Тестирование

```bash
# Юнит-тесты (без PostgreSQL)
make test

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
