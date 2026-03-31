# План реализации web-клиента для GophKeeper

## Цель

Сделать отдельный web-клиент на базе `React + TypeScript + Vite + npm + Ant Design`, который работает с тем же сервером GophKeeper и повторяет пользовательский сценарий существующих `CLI` / `TUI` / `desktop`.

Поведение на старте:
- показывать только 2 действия: `login` и `register`;
- отдельной кнопки `exit` не делать;
- выход из клиента происходит только закрытием вкладки/окна браузера или прерыванием dev-сервера;
- после успешного `register` показать modal `registered successfully` c кнопкой `OK`;
- по нажатию `OK` автоматически выполнить `login` с теми же данными;
- после успешного автологина сразу открыть рабочую сессию.

Поведение после логина:
- поддержать те же пользовательские операции, что уже есть в `cli/tui`:
  - `list`
  - `get`
  - `add`
  - `update`
  - `delete`
  - `sync`
  - `logout`

## На что опираемся

### Что уже есть в проекте

- Сервер уже предоставляет HTTP API для auth, records, sync и uploads:
  - [`internal/api/auth_login_v1_post/handler.go`](/Users/rasabirov/sources/_my/gophkeeper/internal/api/auth_login_v1_post/handler.go)
  - [`internal/api/auth_register_v1_post/handler.go`](/Users/rasabirov/sources/_my/gophkeeper/internal/api/auth_register_v1_post/handler.go)
  - [`internal/api/records_v1_get/handler.go`](/Users/rasabirov/sources/_my/gophkeeper/internal/api/records_v1_get/handler.go)
  - [`internal/api/records_v1_post/handler.go`](/Users/rasabirov/sources/_my/gophkeeper/internal/api/records_v1_post/handler.go)
  - [`internal/api/records_by_id_v1_get/handler.go`](/Users/rasabirov/sources/_my/gophkeeper/internal/api/records_by_id_v1_get/handler.go)
  - [`internal/api/records_by_id_v1_put/handler.go`](/Users/rasabirov/sources/_my/gophkeeper/internal/api/records_by_id_v1_put/handler.go)
  - [`internal/api/records_by_id_v1_delete/handler.go`](/Users/rasabirov/sources/_my/gophkeeper/internal/api/records_by_id_v1_delete/handler.go)
  - [`internal/api/sync_pull_v1_post/handler.go`](/Users/rasabirov/sources/_my/gophkeeper/internal/api/sync_pull_v1_post/handler.go)
  - [`internal/api/sync_push_v1_post/handler.go`](/Users/rasabirov/sources/_my/gophkeeper/internal/api/sync_push_v1_post/handler.go)
  - [`internal/api/uploads_v1_post/handler.go`](/Users/rasabirov/sources/_my/gophkeeper/internal/api/uploads_v1_post/handler.go)
  - [`internal/api/uploads_by_id_v1_get/handler.go`](/Users/rasabirov/sources/_my/gophkeeper/internal/api/uploads_by_id_v1_get/handler.go)

- Готовый desktop-клиент уже реализует нужный UX и может служить шаблоном по структуре экранов:
  - [`cmd/client/desktop/frontend/src/app/App.tsx`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/desktop/frontend/src/app/App.tsx)
  - [`cmd/client/desktop/frontend/src/features/auth/StartScreen.tsx`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/desktop/frontend/src/features/auth/StartScreen.tsx)
  - [`cmd/client/desktop/frontend/src/features/auth/AuthCard.tsx`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/desktop/frontend/src/features/auth/AuthCard.tsx)
  - [`cmd/client/desktop/frontend/src/features/records/Workspace.tsx`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/desktop/frontend/src/features/records/Workspace.tsx)

- В TUI уже есть нужная семантика сценария `register -> success modal -> OK -> auto-login`, и web должен повторить именно ее:
  - [`cmd/client/tui/app/app.go`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/tui/app/app.go)

### Важное архитектурное решение для web

Для web MVP лучше не пытаться переиспользовать `ClientCore` напрямую.

Причины:
- `ClientCore` написан на Go и завязан на локальный cache/store;
- браузер не умеет работать с файловой системой и локальным persistent store так же, как CLI/TUI/desktop;
- для web путь с наименьшим риском и наименьшей связностью это прямой `REST`-клиент в `TypeScript`, который работает с тем же серверным API;
- часть кода из desktop frontend можно копировать и адаптировать, а не выносить в shared-пакет, как и предложено для MVP.

Итого:
- web frontend напрямую вызывает HTTP API сервера;
- auth-состояние и локальный session snapshot хранятся в браузере (`localStorage` / `sessionStorage`);
- бинарные загрузки и скачивания реализуются через browser `File`, `Blob`, `input[type=file]` и download links.

## Предлагаемая структура

### 1. Новый web entrypoint

Добавить новый клиент:
- [`cmd/client/web`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/web)

Ожидаемая структура:
- [`cmd/client/web/index.html`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/web/index.html)
- [`cmd/client/web/package.json`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/web/package.json)
- [`cmd/client/web/vite.config.ts`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/web/vite.config.ts)
- [`cmd/client/web/tsconfig.json`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/web/tsconfig.json)
- [`cmd/client/web/src/main.tsx`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/web/src/main.tsx)
- [`cmd/client/web/src/app/App.tsx`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/web/src/app/App.tsx)
- [`cmd/client/web/src/features/auth`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/web/src/features/auth)
- [`cmd/client/web/src/features/records`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/web/src/features/records)
- [`cmd/client/web/src/shared/api`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/web/src/shared/api)
- [`cmd/client/web/src/shared/types`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/web/src/shared/types)
- [`cmd/client/web/src/shared/lib`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/web/src/shared/lib)
- [`cmd/client/web/src/shared/ui`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/web/src/shared/ui)

### 2. API client слой

Сделать отдельный browser-friendly API слой:
- `http.ts` — общий `fetch` wrapper;
- `auth.ts` — `login/register/logout/refresh`;
- `records.ts` — `list/get/create/update/delete`;
- `sync.ts` — `push/pull/syncNow`;
- `uploads.ts` — upload/download бинарных файлов;
- `storage.ts` — токены, device/client id, базовые настройки, last-used session.

Важно:
- не тащить во frontend Go-модели один-в-один;
- описать свои `DTO` и маппинги;
- централизовать обработку `401/403/409/5xx`;
- поддержать автоматическую подстановку `Authorization` заголовка;
- предусмотреть refresh access token, если web-клиент будет жить дольше одной сессии.

## Экранная модель

### 1. Стартовый экран

Показать только:
- `Login`
- `Register`

Требования:
- без кнопки `Exit`;
- при старте проверять, есть ли сохраненная web-сессия;
- если access token еще валиден, сразу открывать основной экран;
- если access token истек, пробовать `refresh`;
- если refresh не сработал, очищать local session и оставаться на стартовом экране.

### 2. Login

Форма:
- `email`
- `password`
- `Login`
- `Back`

Поведение:
- отправить запрос в `auth/login`;
- сохранить токены и базовую информацию о сессии;
- перейти в рабочий экран;
- при ошибке показать `Alert` или `Modal`.

### 3. Register

Форма:
- `email`
- `password`
- `Register`
- `Back`

Поведение:
- отправить запрос в `auth/register`;
- при успехе показать modal `registered successfully`;
- в modal одна кнопка `OK`;
- по `OK` выполнить `login` с теми же `email/password`;
- при успешном автологине перейти в рабочий экран;
- если автологин не удался, показать ошибку и открыть экран `login` с предзаполненным `email`.

Это поведение должно быть буквально таким же, как в TUI и desktop.

### 4. Основной экран после логина

Для MVP подойдет layout, близкий к desktop:
- верхний header с названием приложения, email пользователя, кнопками `Sync` и `Logout`;
- слева список записей и фильтр;
- справа detail pane выбранной записи;
- сверху toolbar с действиями `Add`, `Update`, `Delete`, `Download file`, `Refresh`.

Доступные сценарии:
- `list` — загрузить и отфильтровать записи;
- `get` — показать полную карточку записи;
- `add` — открыть modal/drawer создания;
- `update` — открыть modal/drawer редактирования;
- `delete` — подтвердить удаление;
- `sync` — синхронизировать данные;
- `logout` — очистить сессию и вернуться на стартовый экран.

## Паритет операций с CLI/TUI

### 1. List

Показывать в таблице:
- `ID`
- `Type`
- `Name`
- `Revision`
- `Deleted`
- preview `Metadata`

Фильтры:
- `all`
- `login`
- `text`
- `binary`
- `card`

Рекомендация:
- использовать `Antd Table` или `List`;
- выделение строки синхронизировать с detail-pane;
- держать отдельную кнопку `Refresh`, даже если после мутаций список и так перечитывается.

### 2. Get

Для небинарных записей:
- подгружать полную запись по `id`;
- показывать структурированные поля в правой панели.

Для бинарных записей:
- показывать тип, revision, metadata, имя файла, размер;
- содержимое файла в UI не рендерить;
- дать действие `Download file`;
- на скачивание получать бинарный контент от сервера и сохранять его через browser download flow.

### 3. Add

Сценарий:
- сначала выбрать тип записи;
- затем показать специализированную форму.

Формы:
- `login`: `name`, `metadata`, `login`, `password`;
- `text`: `name`, `metadata`, `content`;
- `card`: `name`, `metadata`, `number`, `holder`, `expiry`, `cvv`;
- `binary`: `name`, `metadata`, `file`.

Поведение:
- собрать DTO и отправить на сервер;
- для `binary` сначала создать запись, потом загрузить файл;
- после успеха перечитать список и выделить созданную запись.

### 4. Update

Поведение:
- работать от выбранной записи;
- предзаполнять форму текущими значениями;
- разрешить менять `name`, `metadata`, payload;
- для `binary` разрешить выбрать новый файл для замены содержимого;
- после успеха перечитывать список и детали.

### 5. Delete

Поведение:
- удаление только для выбранной записи;
- перед удалением показывать `confirm modal`;
- после успеха перечитывать список;
- если удалили выбранную запись, аккуратно переводить фокус на соседнюю или очищать detail-pane.

### 6. Sync

Поведение:
- отдельная кнопка `Sync`;
- реализация через `push + pull` или готовый sync endpoint, в зависимости от уже используемой серверной модели;
- после успеха перечитывать список и показывать `synced`;
- при конфликте или ошибке показывать понятный user-facing текст.

### 7. Logout

Поведение:
- вызвать `logout` endpoint;
- удалить токены и локальный session snapshot из браузера;
- очистить in-memory state;
- вернуться на стартовый экран.

## DTO и модель данных во frontend

Стоит сразу ввести явные UI DTO:

- `SessionState`
  - `authenticated`
  - `email`
  - `deviceId`
  - `serverAddress`

- `RecordListItem`
  - `id`
  - `type`
  - `name`
  - `revision`
  - `metadataPreview`
  - `deleted`

- `RecordDetails`
  - базовые поля записи;
  - `payloadKind`;
  - flat-поля для конкретного payload;
  - для binary: `fileName`, `size`, `hasContent`.

- `RecordUpsertInput`
  - `id`
  - `type`
  - `name`
  - `metadata`
  - payload-поля;
  - `file?: File | null`.

Это позволит не размазывать knowledge о серверных структурах по компонентам.

## Предлагаемый порядок реализации

### Этап 1. Подготовка web-каркаса

Сделать:
- создать `cmd/client/web`;
- инициализировать `Vite + React + TS`;
- подключить `Ant Design`;
- завести `.env` / `.env.example` для `VITE_API_BASE_URL`;
- добавить npm-скрипты `dev`, `build`, `preview`, `lint`;
- обновить `Makefile` или добавить удобные команды запуска.

Результат:
- поднимается отдельный web dev-server;
- в браузере открывается пустой каркас приложения.

### Этап 2. Базовый HTTP client и session storage

Сделать:
- реализовать `fetch` wrapper;
- централизовать base URL, headers, timeout, JSON parsing;
- добавить хранение `accessToken`, `refreshToken`, `email`, `deviceId`;
- реализовать bootstrap сессии при старте приложения;
- определить формат хранения session state в браузере.

Результат:
- frontend умеет переживать перезагрузку вкладки и восстанавливать сессию.

### Этап 3. Auth flow

Сделать:
- стартовый экран только с `login/register`;
- `login` форму;
- `register` форму;
- success modal `registered successfully`;
- автологин после `OK`;
- возврат в `login` с prefilled email, если автологин после регистрации не удался.

Результат:
- web-клиент полностью закрывает auth UX, идентичный TUI/desktop.

### Этап 4. Shell после логина

Сделать:
- общий layout;
- header с email, `Sync`, `Logout`;
- список записей;
- фильтр по типу;
- detail pane;
- загрузку записей при входе.

Результат:
- уже доступны `list`, `get`, `logout`.

### Этап 5. CRUD для текстовых типов

Сделать:
- формы `add/update` для `login`, `text`, `card`;
- delete confirm;
- перечитывание списка и деталей после мутаций.

Результат:
- закрыт основной пользовательский сценарий для большинства данных.

### Этап 6. Binary flow

Сделать:
- форму создания/обновления бинарной записи;
- upload файла через browser API;
- download файла через `Blob`;
- отображение file metadata в details.

Результат:
- поддержан полный паритет по бинарным записям.

### Этап 7. Sync и polishing

Сделать:
- кнопка `Sync`;
- обработка busy-state на async операциях;
- empty states;
- централизованные уведомления `message` / `modal`;
- улучшение текста ошибок;
- финальная зачистка UI.

Результат:
- web-клиент становится пригодным для повседневного использования.

### Этап 8. Документация и smoke-проверка

Сделать:
- обновить `README`;
- описать запуск web dev/build;
- описать нужные переменные окружения;
- проверить сборку;
- убедиться, что существующие CLI/TUI/desktop не задеты.

## Изменения в инфраструктуре и зависимостях

Понадобится:
- `react`
- `react-dom`
- `typescript`
- `vite`
- `antd`
- возможно `@tanstack/react-query`, если захотим аккуратнее управлять запросами, но для MVP можно обойтись и без него;
- возможно `zod` для валидации DTO/форм, но это не обязательно для первого инкремента.

Также нужно заранее решить:
- будет ли web dev-server ходить в локальный backend через CORS;
- нужен ли proxy в `vite.config.ts`;
- будет ли production web-клиент отдаваться отдельным static hosting или сервером GophKeeper;
- нужна ли отдельная server-side настройка CORS / CSP / cookie policy.

## Тестирование

### Frontend-уровень

Минимально проверить:
- стартовый экран;
- login flow;
- register flow;
- success modal после registration;
- автологин после `OK`;
- list/get;
- add/update/delete по типам;
- binary upload/download;
- sync;
- logout.

### API contract / integration

Проверить руками или тестами:
- корректную сериализацию record payload;
- обработку `401` и refresh flow;
- работу с бинарными upload/download endpoint;
- сценарий при недоступном сервере;
- ошибки валидации сервера.

### Manual smoke

Проверить руками:
- открытие web-клиента без сессии;
- восстановление сессии после reload;
- login;
- register -> `registered successfully` -> `OK` -> auto-login;
- add/get/update/delete для `login`, `text`, `card`, `binary`;
- download binary;
- sync;
- logout;
- повторный вход после logout.

## Риски и решения

### 1. Невозможность прямого переиспользования `ClientCore`

Риск:
- захочется втянуть слишком много Go-кода в browser-сценарий.

Решение:
- держать web отдельным TS-клиентом поверх HTTP API;
- переиспользовать только UX-идеи и часть React-компонентов desktop через copy-adapt.

### 2. CORS и auth в браузере

Риск:
- локальный dev-server может не достучаться до сервера без CORS/proxy;
- токены в браузере требуют аккуратного хранения.

Решение:
- на dev-этапе использовать `vite` proxy;
- для MVP хранить токены в `localStorage`, но держать это как осознанный компромисс;
- отдельно зафиксировать follow-up на более безопасную cookie-схему, если проект пойдет дальше MVP.

### 3. Binary UX в браузере

Риск:
- поведение бинарных файлов отличается от desktop.

Решение:
- опираться на стандартные browser primitives: `File`, `Blob`, `URL.createObjectURL`, hidden file input;
- не пытаться делать file system parity с desktop-клиентом.

### 4. Sync-модель может отличаться от ожиданий web

Риск:
- серверный sync рассчитан на локальный cache/pending queue, а web в MVP может быть более stateless.

Решение:
- на старте реализовать sync как явное серверное действие и полное перечитывание списка;
- если потребуется offline-first поведение, вынести это в отдельный этап после MVP.

### 5. Слишком раннее обобщение между desktop и web

Риск:
- можно увязнуть в выделении shared frontend layers.

Решение:
- копировать desktop frontend куски в web и адаптировать;
- вопрос выделения shared-пакета отложить до появления повторяющейся боли.

## Definition of Done

Задачу можно считать завершенной, когда:
- есть отдельный web-клиент в `cmd/client/web`;
- клиент запускается через `npm` / `vite`;
- на старте отображаются только `login` и `register`;
- после `register` показывается modal `registered successfully` с кнопкой `OK`;
- `OK` делает автологин теми же данными;
- после логина доступны `list`, `get`, `add`, `update`, `delete`, `sync`, `logout`;
- бинарные файлы можно загрузить и скачать;
- `logout` возвращает на стартовый экран;
- документирован запуск web-клиента и его конфигурация.

## Рекомендуемый первый инкремент

Самый безопасный первый кусок реализации:
- поднять `Vite + React + Antd` каркас;
- сделать `auth API client` и session storage;
- реализовать только `session restore`, `login`, `register`, success modal и `logout`;
- после этого переходить к records workspace и CRUD.

Так мы быстро проверим всю web-цепочку `browser -> HTTP API -> server`, не начиная сразу с более дорогой части про формы всех типов записей и binary flow.
