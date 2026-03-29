# Dependency Graph — gophkeeper

```mermaid
graph TD
    classDef completed fill:lightgreen
    classDef crit fill:orange
    subgraph P1["Phase 1 — Каркас"]
        T1["<b>task_1</b><br/>Каркас репозитория"]
        T2["<b>task_2</b><br/>Инженерная обвязка"]
        T3["<b>task_3</b><br/>Доменная модель"]
    end
    class P1 completed
    class T1 completed
    class T2 completed
    class T3 completed

    subgraph P3["Phase 3 — Контракты и transport"]
        T4["<b>task_4</b><br/>gRPC контракты"]
        T5["<b>task_5</b><br/>HTTP auth endpoints"]
        T6["<b>task_6</b><br/>HTTP records endpoints"]
        T7["<b>task_7</b><br/>HTTP sync/uploads/health"]
    end
    class P3 completed
    class T4 completed
    class T5 completed
    class T6 completed
    class T7 completed

    subgraph P4["Phase 4 — Инфраструктура"]
        T8["<b>task_8</b><br/>Bootstrap сервера, TLS"]
        T9["<b>task_9</b><br/>PostgreSQL, миграции"]
    end
    class P4 completed
    class T8 completed
    class T9 completed

    subgraph P5["Phase 5 — Безопасность"]
        T10["<b>task_10</b><br/>Криптография, key mgmt"]
    end
    class P5 completed
    class T10 completed

    subgraph P6["Phase 6 — Use-cases"]
        T11["<b>task_11</b><br/>Auth use-case"]
        T12["<b>task_12</b><br/>Records use-case"]
        T13["<b>task_13</b><br/>Uploads use-case"]
    end
    class T11 completed
    class T12 completed

    subgraph P7["Phase 7 — Синхронизация"]
        T14["<b>task_14</b><br/>Sync use-case"]
    end
    class T14 crit

    subgraph P8["Phase 8 — Client core"]
        T15["<b>task_15</b><br/>Shared client core"]
    end
    class T15 crit

    subgraph P9["Phase 9 — CLI"]
        T16["<b>task_16</b><br/>CLI-клиент"]
    end
    class T16 crit

    subgraph P10["Phase 10 — Финализация"]
        T17["<b>task_17</b><br/>Тестирование и CI"]
        T18["<b>task_18</b><br/>Документация, MVP"]
    end
    class T17 crit
    class T18 crit

    %% Phase 1
    T1 --> T2
    T1 --> T3

    %% Phase 3
    T2 --> T4
    T3 --> T4
    T2 --> T5
    T3 --> T5
    T2 --> T6
    T3 --> T6
    T2 --> T7
    T3 --> T7

    %% Phase 4
    T4 --> T8
    T5 --> T8
    T6 --> T8
    T7 --> T8
    T3 --> T9

    %% Phase 5
    T3 --> T10
    T9 --> T10

    %% Phase 6
    T4 --> T11
    T5 --> T11
    T8 --> T11
    T9 --> T11
    T10 --> T11

    T4 --> T12
    T6 --> T12
    T8 --> T12
    T9 --> T12
    T10 --> T12
    T11 --> T12

    T4 --> T13
    T7 --> T13
    T8 --> T13
    T9 --> T13
    T10 --> T13
    T11 --> T13

    %% Phase 7
    T4 --> T14
    T7 --> T14
    T8 --> T14
    T9 --> T14
    T11 --> T14
    T12 --> T14
    T13 --> T14

    %% Phase 8
    T4 --> T15
    T11 --> T15
    T12 --> T15
    T13 --> T15
    T14 --> T15

    %% Phase 9
    T15 --> T16

    %% Phase 10
    T2 --> T17
    T4 --> T17
    T8 --> T17
    T9 --> T17
    T10 --> T17
    T11 --> T17
    T12 --> T17
    T13 --> T17
    T14 --> T17
    T15 --> T17
    T16 --> T17

    T2 --> T18
    T3 --> T18
    T10 --> T18
    T16 --> T18
    T17 --> T18
```

## Критический путь

```
task_1 → task_3 → task_9 → task_10 → task_11 → task_12 → task_14 → task_15 → task_16 → task_17 → task_18
```

## Параллельные группы (можно выполнять одновременно)

| Группа | Задачи |
|--------|--------|
| 1 | <span style="background-color: lightgreen;">task_1</span> |
| 2 | <span style="background-color: lightgreen;">task_2</span>, <span style="background-color: lightgreen;">task_3</span> |
| 3 | <span style="background-color: lightgreen;">task_4</span>, <span style="background-color: lightgreen;">task_5</span>, <span style="background-color: lightgreen;">task_6</span>, <span style="background-color: lightgreen;">task_7</span> |
| 4 | <span style="background-color: lightgreen;">task_8</span>, <span style="background-color: lightgreen;">task_9</span> |
| 5 | <span style="background-color: lightgreen;">task_10</span> |
| 6 | <span style="background-color: lightgreen;">task_11</span> |
| 7 | <span style="background-color: lightgreen;">task_12</span>, task_13 |
| 8 | task_14 |
| 9 | task_15 |
| 10 | task_16 |
| 11 | task_17, task_18 |
