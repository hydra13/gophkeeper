# Machine Map

```yaml
tasks:
  - id: task_1
    title: "Каркас репозитория и MVP-границы"
    file: ".memory-bank/tasks/task_1.md"
    status: "completed"
    phase: 1
    parallel_group: 1
    mvp: true
    depends_on: []
    blocks: [task_2, task_3]
    notes: "Стартовая structural-задача. Фиксирует layout и post-MVP заглушки."

  - id: task_2
    title: "Инженерная обвязка и базовые команды проекта"
    file: ".memory-bank/tasks/task_2.md"
    status: "completed"
    phase: 1
    parallel_group: 2
    mvp: true
    depends_on: [task_1]
    blocks: [task_17, task_18]
    notes: "Создает стабильную базу для генерации, линтинга, тестов и сборки."

  - id: task_3
    title: "Доменная модель, инварианты синхронизации и загрузок"
    file: ".memory-bank/tasks/task_3.md"
    status: "completed"
    phase: 2
    parallel_group: 3
    mvp: true
    depends_on: [task_1]
    blocks: [task_4, task_5, task_6, task_7, task_9, task_10, task_11, task_12, task_13, task_14, task_18]
    notes: "Опора для API, persistence, crypto и client core."

  - id: task_4
    title: "gRPC контракты в rpc/proto/v1"
    file: ".memory-bank/tasks/task_4.md"
    status: "completed"
    phase: 3
    parallel_group: 4
    mvp: true
    depends_on: [task_2, task_3]
    blocks: [task_8, task_11, task_12, task_13, task_14, task_17]
    notes: "Фиксирует versioned контракты и генерацию protobuf."

  - id: task_5
    title: "HTTP auth endpoints в api/"
    file: ".memory-bank/tasks/task_5.md"
    status: "completed"
    phase: 3
    parallel_group: 4
    mvp: true
    depends_on: [task_2, task_3]
    blocks: [task_8, task_11]
    notes: "Изолированная подготовка HTTP auth transport слоя."

  - id: task_6
    title: "HTTP records endpoints в api/"
    file: ".memory-bank/tasks/task_6.md"
    status: "completed"
    phase: 3
    parallel_group: 4
    mvp: true
    depends_on: [task_2, task_3]
    blocks: [task_8, task_12]
    notes: "CRUD transport-контракты и тестовый каркас."

  - id: task_7
    title: "HTTP sync, uploads и health endpoints в api/"
    file: ".memory-bank/tasks/task_7.md"
    status: "completed"
    phase: 3
    parallel_group: 4
    mvp: true
    depends_on: [task_2, task_3]
    blocks: [task_8, task_13, task_14]
    notes: "Sync и uploads выделены отдельно из-за повышенной сложности."

  - id: task_8
    title: "Конфигурация, bootstrap сервера и TLS-only запуск"
    file: ".memory-bank/tasks/task_8.md"
    status: "completed"
    phase: 4
    parallel_group: 5
    mvp: true
    depends_on: [task_4, task_5, task_6, task_7]
    blocks: [task_11, task_12, task_13, task_14, task_17]
    notes: "Инфраструктурный bootstrap после стабилизации transport-каркаса. Строится через интерфейсы и не требует полной готовности business/use-case слоев."

  - id: task_9
    title: "PostgreSQL, файловое хранилище и миграции"
    file: ".memory-bank/tasks/task_9.md"
    status: "completed"
    phase: 4
    parallel_group: 5
    mvp: true
    depends_on: [task_3]
    blocks: [task_10, task_11, task_12, task_13, task_14, task_17]
    notes: "Persistence-фундамент для auth, records, sync, uploads и key management."

  - id: task_10
    title: "Криптография, key management и re-encryption"
    file: ".memory-bank/tasks/task_10.md"
    status: "completed"
    phase: 5
    parallel_group: 6
    mvp: true
    depends_on: [task_3, task_9]
    blocks: [task_11, task_12, task_13, task_14, task_17, task_18]
    notes: "Core-безопасность MVP должна быть реализована до прикладных сценариев."

  - id: task_11
    title: "Auth use-case, tokens и device-aware sessions"
    file: ".memory-bank/tasks/task_11.md"
    status: "completed"
    phase: 6
    parallel_group: 7
    mvp: true
    depends_on: [task_4, task_5, task_8, task_9, task_10]
    blocks: [task_14, task_15, task_16, task_17]
    notes: "Общий auth use-case поверх готовых transport-контрактов, bootstrap, persistence и crypto."

  - id: task_12
    title: "Use-case слой записей и gRPC/HTTP интеграция"
    file: ".memory-bank/tasks/task_12.md"
    status: "completed"
    phase: 6
    parallel_group: 8
    mvp: true
    depends_on: [task_4, task_6, task_8, task_9, task_10, task_11]
    blocks: [task_14, task_15, task_17]
    notes: "Повторная валидация пройдена: records use-case подключен к HTTP и gRPC, mapping ошибок и soft delete работают корректно, binary transport-контракт приведен к модели metadata + attachment reference через payload_version без raw `binary.data` в CRUD."

  - id: task_13
    title: "Use-case слой uploads и бинарных вложений"
    file: ".memory-bank/tasks/task_13.md"
    status: "completed"
    phase: 6
    parallel_group: 8
    mvp: true
    depends_on: [task_4, task_7, task_8, task_9, task_10, task_11]
    blocks: [task_14, task_15, task_17]
    notes: "Повторная валидация пройдена: замечания закрыты, все критерии приёмки выполнены."

  - id: task_14
    title: "Sync use-case и разрешение конфликтов"
    file: ".memory-bank/tasks/task_14.md"
    status: "completed"
    phase: 7
    parallel_group: 9
    mvp: true
    depends_on: [task_4, task_7, task_8, task_9, task_11, task_12, task_13]
    blocks: [task_15, task_16, task_17]
    notes: "Повторная валидация 2026-03-29 пройдена: все 6 критериев приёмки выполнены. Исправлены проверка BaseRevision в pushDelete, сохранение снимков конфликтов в БД, восстановление soft-deleted записи через update и интеграционные сценарии stale delete/delete-vs-update/delete-restore."

  - id: task_15
    title: "Shared client core и локальный кеш"
    file: ".memory-bank/tasks/task_15.md"
    status: "completed"
    phase: 8
    parallel_group: 10
    mvp: true
    depends_on: [task_4, task_11, task_12, task_13, task_14]
    blocks: [task_16, task_17]
    notes: "Повторная валидация 2026-03-29 пройдена: CLI подключен через clientcore, resume upload/download восстановлен и покрыт тестами, package-level границы apiclient/clientcore/cache задокументированы."

  - id: task_16
    title: "CLI-клиент в cmd/client/cli"
    file: ".memory-bank/tasks/task_16.md"
    status: "completed"
    phase: 9
    parallel_group: 11
    mvp: true
    depends_on: [task_15]
    blocks: [task_17, task_18]
    notes: "Повторная валидация 2026-03-29 пройдена: замечания закрыты, все 6 критериев приёмки выполнены, CLI smoke/e2e-покрытие подтверждено тестами entrypoint'ов и бинарными проверками version/ошибок запуска."

  - id: task_17
    title: "Тестирование и CI"
    file: ".memory-bank/tasks/task_17.md"
    status: "completed"
    phase: 10
    parallel_group: 12
    mvp: true
    depends_on: [task_2, task_4, task_8, task_9, task_10, task_11, task_12, task_13, task_14, task_15, task_16]
    blocks: [task_18]
    notes: "Повторная валидация 2026-03-30 пройдена: замечания закрыты, все критерии приёмки выполнены; test matrix, coverage и автоматизация проверок подтверждены."

  - id: task_18
    title: "Документация и выпуск MVP"
    file: ".memory-bank/tasks/task_18.md"
    status: "completed"
    phase: 10
    parallel_group: 13
    mvp: true
    depends_on: [task_2, task_3, task_10, task_16, task_17]
    blocks: []
    notes: "Повторная валидация 2026-03-30 пройдена: замечания закрыты, все критерии приёмки выполнены; README и backlog-границы актуализированы, package docs добавлены, runbook'и по key rotation/re-encryption уточнены, release readiness review зафиксирован."

  - id: task_19
    title: "Поддержка metadata в CLI для всех типов записей"
    file: ".memory-bank/tasks/task_19.md"
    status: "completed"
    phase: 11
    parallel_group: 14
    mvp: true
    depends_on: [task_16, task_18]
    blocks: [task_22, task_23]
    notes: "Повторная валидация 2026-03-30 пройдена: замечания закрыты, все критерии приёмки выполнены; metadata поддержана в add/update/get/list, help и README обновлены, CLI и e2e roundtrip через server/sync подтверждены тестами."

  - id: task_20
    title: "Согласовать CLI с TLS-only сервером и воспроизводимым quick start"
    file: ".memory-bank/tasks/task_20.md"
    status: "completed"
    phase: 11
    parallel_group: 14
    mvp: true
    depends_on: [task_8, task_15, task_16, task_18]
    blocks: [task_22, task_23]
    notes: "Повторная валидация 2026-03-30 пройдена: замечания закрыты, все 7 критериев приёмки выполнены; CLI fail-fast требует TLS cert, quick start и default config синхронизированы на configs/config.dev.json, documented TLS-path с configs/certs/dev.crt подтверждён тестами."

  - id: task_21
    title: "Fail-fast bootstrap сервера без stub-режима"
    file: ".memory-bank/tasks/task_21.md"
    status: "completed"
    phase: 11
    parallel_group: 14
    mvp: true
    depends_on: [task_8, task_9, task_18]
    blocks: [task_22, task_23]
    notes: "Повторная валидация 2026-03-30 пройдена: замечания закрыты, все 7 критериев приёмки выполнены; bootstrap fail-fast требует database.dsn, завершает старт при ошибке подключения/миграций, README синхронизирован, поведение health/readiness зафиксировано, тесты cmd/server и internal/config проходят."

  - id: task_22
    title: "Подтвердить и обеспечить покрытие тестами не ниже 70% по всей системе"
    file: ".memory-bank/tasks/task_22.md"
    status: "pending"
    phase: 12
    parallel_group: 15
    mvp: true
    depends_on: [task_17, task_19, task_20, task_21]
    blocks: [task_23]
    notes: "Делает coverage gate доказуемым: единая метрика, честный подсчёт по системе и подтверждённый порог >= 70%."

  - id: task_23
    title: "Повторная валидация соответствия MVP и release readiness"
    file: ".memory-bank/tasks/task_23.md"
    status: "pending"
    phase: 12
    parallel_group: 16
    mvp: true
    depends_on: [task_19, task_20, task_21, task_22]
    blocks: []
    notes: "Финализирует remediation-цикл: обновляет validation/release-readiness и подтверждает, что исправления действительно закрыли найденные расхождения."
```
