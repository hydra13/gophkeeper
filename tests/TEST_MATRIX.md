# Test Matrix — GophKeeper

## Область тестирования

| Область         | Пакеты                                                    | Unit | Integration | E2E |
|-----------------|-----------------------------------------------------------|------|-------------|-----|
| **Domain**      | `internal/models`                                         | +    | -           | -   |
| **Auth**        | `internal/services/auth`, `internal/services/passwords`   | +    | -           | -   |
| **CRUD Records**| `internal/services/records`, `internal/api/records_*`     | +    | -           | -   |
| **Uploads**     | `internal/services/uploads`, `internal/api/uploads_*`     | +    | + (DB)      | -   |
| **Sync**        | `internal/services/sync`, `internal/api/sync_*`           | +    | + (DB)      | -   |
| **TLS**         | `internal/middlewares` (tls.go)                           | +    | -           | -   |
| **Key Rotation**| `internal/services/keys`, `internal/services/crypto`      | +    | -           | -   |
| **Re-encrypt**  | `internal/jobs/reencrypt`                                 | -    | + (DB)      | -   |
| **gRPC**        | `internal/rpc`                                            | +    | -           | -   |
| **Migrations**  | `internal/migrations`, `migrations/`                      | -    | + (DB)      | -   |
| **Config**      | `internal/config`                                         | +    | -           | -   |
| **Storage**     | `internal/storage`                                        | +    | -           | -   |
| **Client Core** | `pkg/clientcore`, `pkg/cache`                             | +    | -           | -   |
| **CLI**         | `cmd/client/cli`                                          | +    | -           | +   |

## Unit Tests

### models
- [x] Record: Validate, BumpRevision, Restore, IsDeleted
- [x] Session: IsExpired, Touch
- [x] KeyVersion: IsDecryptionCapable
- [x] SyncConflict: Resolve
- [x] Upload models: UploadSession, DownloadSession, Chunk transitions
- [x] Payload: LoginPayload, TextPayload, BinaryPayload, CardPayload

### auth
- [x] Auth service: Register, Login, Refresh, Logout
- [x] JWT generation and validation
- [x] Password hashing and verification

### records
- [x] Records service: Create, Get, List, Update, Delete
- [x] Validation: empty name, nil payload, invalid type, key version

### uploads
- [x] Upload service: CreateSession, UploadChunk, Complete, Download
- [x] Chunk order validation, duplicate detection

### sync
- [x] Push: create, update, delete with conflict detection
- [x] Pull: pagination, conflict inclusion
- [x] GetConflicts, ResolveConflict

### crypto / keys
- [x] Encryption/decryption roundtrip
- [x] Key manager: generate, rotate, get active version

### gRPC transport
- [x] Auth: Register, Login, Refresh, Logout
- [x] Data: CreateRecord, GetRecord, ListRecords, UpdateRecord, DeleteRecord
- [x] Sync: Push, Pull, GetConflicts, ResolveConflict
- [x] Uploads: CreateSession, UploadChunk, DownloadChunk
- [x] Health: HealthCheck

### HTTP transport
- [x] Auth endpoints: login, register, refresh, logout
- [x] Records CRUD endpoints
- [x] Sync push/pull endpoints
- [x] Upload endpoints
- [x] Health endpoint

### middleware
- [x] Auth middleware: token extraction, validation
- [x] gRPC interceptor: metadata extraction

### config
- [x] Validate: required fields, TLS files existence
- [x] Load: file loading, env overrides, flags

### storage
- [x] LocalBlob: Save, Read, Delete, Exists

### client
- [x] clientcore: Core operations
- [x] cache: MemoryCache, JSONCache

## Integration Tests

> Требуют running PostgreSQL (`GK_TEST_DATABASE_DSN`)

- [x] Database repository: CRUD operations
- [x] Sync service: full sync flow
- [x] Uploads service: chunk upload/download with storage
- [x] Re-encrypt job: key rotation with data re-encryption

## Transport Contract Parity

> Проверка паритета обязательных сценариев между HTTP и gRPC транспортами

- [x] Transport parity table: `tests/transport_parity_test.go`
- [x] Все обязательные операции покрыты: auth (4), records (5), sync (2), uploads (3), health (1)
- [x] Каждый сценарий имеет привязку к тестам обоих транспортов и ожидаемые коды ошибок

## E2E Tests

> **Post-MVP scope.** E2E-тесты требуют запущенного сервера с БД и TLS и не входят в MVP task_17.
> MVP покрывается unit + integration тестами. E2E вынесены в отдельную итерацию.

- [ ] CLI register/login/list/add/get/sync flow *(post-MVP)*
- [ ] Full sync roundtrip between two devices *(post-MVP)*
- [ ] Upload/download binary data *(post-MVP)*
- [ ] Re-encryption after key rotation *(post-MVP)*
