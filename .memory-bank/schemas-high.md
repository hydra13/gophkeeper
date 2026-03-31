# Верхнеуровневые схемы архитектуры GophKeeper

Ниже несколько Mermaid-представлений одной и той же верхнеуровневой архитектуры.
Во всех вариантах отражены:

- клиенты: `CLI`, `TUI`, `Desktop`
- `Server`
- `PostgreSQL`
- `S3-compatible Blob Storage` / `MinIO`

## Вариант 1. Контекстная схема

Самый простой и читаемый вариант для README, overview и презентаций.

```mermaid
flowchart LR
    subgraph Clients["Clients"]
        CLI["CLI Client"]
        TUI["TUI Client"]
        Desktop["Desktop Client"]
    end

    Server["GophKeeper Server"]
    DB[("PostgreSQL")]
    S3[("S3 / MinIO Blob Storage")]

    CLI -->|"TLS / gRPC"| Server
    TUI -->|"TLS / gRPC"| Server
    Desktop -->|"TLS / gRPC"| Server

    Server -->|"metadata, users, sessions,\nrecord revisions"| DB
    Server -->|"binary payloads,\nchunk upload/download"| S3
```

## Вариант 2. Слои и общие компоненты

Подходит, если хочется подчеркнуть, что разные клиенты используют общую клиентскую логику, а сервер разделён на transport/service/storage слои.

```mermaid
flowchart TB
    subgraph ClientApps["Client Applications"]
        CLI["CLI"]
        TUI["TUI"]
        Desktop["Desktop"]
    end

    subgraph ClientShared["Shared Client Layer"]
        Core["ClientCore"]
        Cache["Local Cache"]
    end

    subgraph ServerApp["Server"]
        API["HTTP / gRPC Transport"]
        Services["Business Services"]
        Repos["Repositories / Storage Adapters"]
    end

    DB[("PostgreSQL")]
    S3[("S3 / MinIO")]

    CLI --> Core
    TUI --> Core
    Desktop --> Core

    Core <--> Cache
    Core -->|"TLS / gRPC"| API

    API --> Services
    Services --> Repos
    Repos --> DB
    Repos --> S3
```

## Вариант 3. Потоки данных по типам

Более наглядный вариант, если важно отделить обычные записи от бинарных файлов.

```mermaid
flowchart LR
    subgraph Clients["Clients"]
        CLI["CLI"]
        TUI["TUI"]
        Desktop["Desktop"]
    end

    Server["GophKeeper Server"]
    DB[("PostgreSQL")]
    S3[("S3 / MinIO")]

    Clients -->|"auth, CRUD, sync,\nrecord metadata"| Server
    Clients -->|"binary upload/download\nvia chunk API"| Server

    Server -->|"users, sessions,\nrecords, revisions,\nkey versions"| DB
    Server -->|"encrypted binary blobs"| S3
```

## Вариант 4. Deployment view

Полезен для infra-обсуждений и описания runtime-размещения компонентов.

```mermaid
flowchart TB
    subgraph UserDevice["User Device"]
        CLI["CLI Binary"]
        TUI["TUI Binary"]
        Desktop["Desktop App"]
        Cache["Local Cache Files"]
    end

    subgraph BackendHost["Backend Environment"]
        Server["GophKeeper Server"]
        DB[("PostgreSQL")]
        S3[("S3-compatible Storage / MinIO")]
    end

    CLI -.-> Cache
    TUI -.-> Cache
    Desktop -.-> Cache

    CLI -->|"TLS / gRPC"| Server
    TUI -->|"TLS / gRPC"| Server
    Desktop -->|"TLS / gRPC"| Server

    Server --> DB
    Server --> S3
```

## Вариант 5. Самый компактный для README

Если нужна короткая схема без внутренних деталей.

```mermaid
flowchart LR
    CLI["CLI"]
    TUI["TUI"]
    Desktop["Desktop"]
    Server["Server"]
    DB[("PostgreSQL")]
    S3[("S3 / MinIO")]

    CLI --> Server
    TUI --> Server
    Desktop --> Server
    Server --> DB
    Server --> S3
```

## Какой вариант когда использовать

- `Вариант 1` — лучший общий overview
- `Вариант 2` — лучший для архитектурного описания кода
- `Вариант 3` — лучший для объяснения различия metadata и binary flow
- `Вариант 4` — лучший для deployment/runtime описания
- `Вариант 5` — лучший для краткого README-блока
