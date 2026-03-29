# gophkeeper

Менеджер паролей GophKeeper.

MVP включает только `server + cli`. Каталоги `cmd/client/desktop` и `cmd/client/web` пока остаются как backlog-заглушки и не входят в первую поставку.

## Локальная разработка

### Требования

- Go 1.25+
- protoc + protoc-gen-go + protoc-gen-go-grpc
- golangci-lint
- goimports

### Команды Makefile

| Команда            | Описание                                      |
| ------------------ | --------------------------------------------- |
| `make fmt`         | Форматирование кода через goimports           |
| `make lint`        | Статический анализ (golangci-lint)            |
| `make test`        | Запуск тестов с race detector и покрытием     |
| `make cover`       | Генерация HTML-отчёта покрытия                |
| `make proto`       | Генерация Go-кода из protobuf (rpc/proto/v1/) |
| `make build`       | Сборка server + client                        |
| `make build-server`| Сборка только сервера                         |
| `make build-client`| Сборка только CLI-клиента                     |
| `make clean`       | Удаление артефактов сборки                    |

### Конфигурация

Конфигурация загружается из JSON-файла. Шаблон: `configs/config.example.json`.

```sh
cp configs/config.example.json configs/config.local.json
```

#### Поля конфигурации

| Поле                    | Env                       | Описание                            |
| ----------------------- | ------------------------- | ----------------------------------- |
| `server.address`        | `GK_SERVER_ADDRESS`       | Адрес HTTP-сервера (по умолчанию `:8080`) |
| `server.grpc_address`   | `GK_GRPC_ADDRESS`         | Адрес gRPC-сервера (по умолчанию `:9090`) |
| `server.tls_cert_file`  | `GK_TLS_CERT_FILE`        | Путь к TLS-сертификату              |
| `server.tls_key_file`   | `GK_TLS_KEY_FILE`         | Путь к TLS-ключу                    |
| `database.dsn`          | `GK_DATABASE_DSN`         | DSN подключения к PostgreSQL        |
| `auth.jwt_secret`       | `GK_JWT_SECRET`           | Секрет для подписи JWT              |
| `auth.token_duration`   | `GK_TOKEN_DURATION`       | Срок жизни токена (по умолчанию `24h`) |
| `crypto.master_key`     | `GK_MASTER_KEY`           | Мастер-ключ шифрования данных       |

Все соединения работают только через TLS.

### База данных

Для развёртывания PostgreSQL используется Docker:

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

DSN для подключения: `postgres://postgres:postgres@localhost:5432/gophkeeper?sslmode=disable`

Для запуска интеграционных тестов:

```bash
GK_TEST_DATABASE_DSN="postgres://postgres:postgres@localhost:5432/gophkeeper?sslmode=disable" make test
```
