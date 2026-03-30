# План реализации desktop-клиента для GophKeeper

## Цель

Сделать отдельный desktop-клиент на базе `Wails` с фронтендом `React + TypeScript + Vite` и UI-компонентами `Ant Design`, который работает с тем же сервером и тем же `ClientCore`, что и текущие `CLI` и `TUI`.

Поведение на старте:
- показать только 2 действия: `login` и `register`;
- штатного пункта `exit` не делать;
- выход из приложения должен происходить только через закрытие окна или прерывание процесса;
- после успешного `register` показать modal `registered successfully` с кнопкой `OK`;
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

## Что уже есть и что стоит переиспользовать

### Готовые опорные слои

- Общая клиентская логика уже собрана в [`pkg/clientcore/core.go`](/Users/rasabirov/sources/_my/gophkeeper/pkg/clientcore/core.go).
- Общий bootstrap для фронтендов уже вынесен в [`cmd/client/common/bootstrap.go`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/common/bootstrap.go).
- Общие UI/helper-функции для типов записей, payload и бинарных файлов уже лежат в [`pkg/clientui/records.go`](/Users/rasabirov/sources/_my/gophkeeper/pkg/clientui/records.go).
- TUI уже реализует нужный UX регистрации:
  - [`cmd/client/tui/app/app.go`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/tui/app/app.go)
  - сценарий `register -> modal "registered successfully" -> OK -> auto-login`.

### Вывод для desktop

Desktop-клиент не должен дублировать бизнес-логику CLI/TUI.
Правильная граница ответственности:
- `ClientCore` отвечает за auth, CRUD, sync и работу с бинарными данными;
- `pkg/clientui` отвечает за общие клиентские преобразования и вспомогательные операции;
- `Wails`-слой отвечает за orchestration, state и вызовы Go backend из React UI.

## Предлагаемая структура

### 1. Новый desktop entrypoint

Добавить отдельный клиент:
- [`cmd/client/desktop`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/desktop)

Ожидаемая структура:
- [`cmd/client/desktop/main.go`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/desktop/main.go) — запуск Wails app;
- [`cmd/client/desktop/app.go`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/desktop/app.go) — lifecycle и регистрация backend-сервисов;
- [`cmd/client/desktop/backend`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/desktop/backend) — Go API, доступный фронтенду;
- [`cmd/client/desktop/frontend`](/Users/rasabirov/sources/_my/gophkeeper/cmd/client/desktop/frontend) — React/Vite приложение.

### 2. Go backend для Wails

Сделать backend-слой, который оборачивает `ClientCore` и отдаёт фронтенду простое API.

Примерная ответственность:
- `SessionService`
  - `GetSessionState()`
  - `Login(email, password)`
  - `Register(email, password)`
  - `Logout()`
- `RecordsService`
  - `ListRecords(filter)`
  - `GetRecord(id)`
  - `CreateRecord(input)`
  - `UpdateRecord(input)`
  - `DeleteRecord(id)`
  - `SyncNow()`
- `BinaryService`
  - `PickFileForUpload()`
  - `SaveBinaryAs(recordID)`
  - `UploadBinary(recordID, path)`
  - `DownloadBinary(recordID, savePath)`

Важно:
- frontend не должен знать про `models.RecordPayload` как внутреннюю Go-структуру;
- backend должен вернуть удобные DTO для UI;
- ошибки нужно нормализовать в понятные user-facing сообщения.

### 3. React frontend

Фронтенд на `React + TS + Vite + Ant Design` стоит разделить так:
- `src/app` — инициализация приложения, routing/state shell;
- `src/features/auth` — login/register/start screens;
- `src/features/records` — list/detail/forms/delete flow;
- `src/features/sync` — sync action и статус;
- `src/shared/api` — вызовы Wails-generated bindings;
- `src/shared/types` — DTO и UI-модели;
- `src/shared/ui` — layout, status bar, common modals/forms;
- `src/shared/lib` — mapping, formatters, validation.

## Экранная модель

### 1. Стартовый экран

Показать экран выбора из двух действий:
- `Login`
- `Register`

Требования:
- без отдельной кнопки `Exit`;
- при старте проверять, восстановилась ли сохранённая сессия через `core.RestoreAuth()` из bootstrap;
- если сессия уже валидна, сразу открывать основной экран;
- если сессии нет, оставаться на стартовом экране.

UI-вариант:
- центральная карточка;
- логотип/название приложения;
- 2 primary/secondary action buttons;
- краткая подсказка про подключение к серверу и локальный кэш.

### 2. Login

Форма:
- `email`
- `password`
- `Login`
- `Back`

Поведение:
- по `Login` вызвать Go backend -> `core.Login`;
- при успехе загрузить рабочий экран;
- при ошибке показать `Antd Modal` или `Alert`;
- `Back` возвращает на стартовый экран.

### 3. Register

Форма:
- `email`
- `password`
- `Register`
- `Back`

Поведение:
- по `Register` вызвать `core.Register`;
- при успехе показать modal с текстом `registered successfully`;
- в modal одна кнопка `OK`;
- по `OK` выполнить `core.Login` с теми же `email/password`;
- при успешном автологине перейти в рабочую сессию;
- если автологин не удался, показать ошибку и открыть экран `login` с предзаполненным `email`.

Это поведение надо держать идентичным TUI, а не придумывать новую семантику.

### 4. Основной экран после логина

Рекомендуемый MVP-layout:
- верхний header с названием приложения, email пользователя, кнопками `Sync` и `Logout`;
- слева список записей и фильтр;
- справа detail-pane по выбранной записи;
- сверху или сбоку toolbar с действиями `Add`, `Update`, `Delete`, `Save file`, `Refresh`.

Доступные сценарии:
- `list` — загрузить и отфильтровать записи;
- `get` — показать полную карточку выбранной записи;
- `add` — открыть modal/drawer создания;
- `update` — открыть modal/drawer редактирования;
- `delete` — подтвердить удаление;
- `sync` — синхронизировать данные;
- `logout` — завершить сессию и вернуться на стартовый экран.

## Маппинг CLI/TUI сценариев в desktop

### 1. List

Источник:
- `core.ListRecords(ctx, recordType)`

Что показывать в таблице:
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

Рекомендация по UI:
- использовать `Antd Table`;
- выделение строки синхронизировать с detail-pane;
- отдельная кнопка `Refresh`, хотя `ListRecords` и так может подтягивать новые данные.

### 2. Get

Для небинарных записей:
- грузить `core.GetRecord`;
- показывать структурированные поля в detail-pane.

Для бинарных записей:
- в detail-pane показывать только метаданные, тип, revision и размер;
- содержимое файла в UI не рендерить;
- дать пользователю действие `Save file...`;
- на сохранение вызвать `core.DownloadBinary(...)` и записать файл по выбранному пути.

### 3. Add

Сценарий:
- сначала выбрать тип записи;
- затем открыть специализированную форму.

Формы:
- `login`: `name`, `metadata`, `login`, `password`;
- `text`: `name`, `metadata`, `content`;
- `card`: `name`, `metadata`, `number`, `holder`, `expiry`, `cvv`;
- `binary`: `name`, `metadata`, `file path / file picker`.

Поведение:
- payload строить через shared helper из `pkg/clientui`;
- запись создавать через `core.SaveRecord`;
- бинарный файл после создания загружать через `core.UploadBinary`;
- после успеха обновлять таблицу и выделять созданную запись.

### 4. Update

Поведение:
- работать от выбранной записи;
- предзаполнять форму текущими значениями;
- разрешить менять `name`, `metadata`, payload;
- для `binary` разрешить указать новый файл для замены содержимого;
- использовать `core.SaveRecord`;
- если бинарный payload обновлён, вызывать `core.UploadBinary`.

### 5. Delete

Поведение:
- удаление только для выбранной записи;
- перед удалением показывать `confirm modal`;
- затем вызывать `core.DeleteRecord`;
- после успеха обновлять список и detail-pane.

### 6. Sync

Поведение:
- кнопка `Sync`;
- вызов `core.SyncNow`;
- показать явный статус `synced` или ошибку;
- после успеха перечитать список записей.

### 7. Logout

Поведение:
- вызвать `core.Logout`;
- очистить локальный UI-state;
- вернуться на стартовый экран с `login/register`.

## DTO и граница между Go и React

Чтобы не тянуть внутренние Go-модели напрямую во frontend, стоит ввести простые DTO:

- `SessionStateDTO`
  - `authenticated`
  - `email`
  - `deviceId`
- `RecordListItemDTO`
  - `id`
  - `type`
  - `name`
  - `revision`
  - `metadataPreview`
  - `deleted`
- `RecordDetailsDTO`
  - базовые поля записи;
  - отдельный `payloadKind`;
  - payload-поля в явном flat-формате;
  - для binary: `size`, `hasContent`.
- `UpsertRecordDTO`
  - `id`
  - `type`
  - `name`
  - `metadata`
  - payload-поля;
  - опциональный `filePath` для binary.

Это позволит фронтенду жить независимо от деталей `internal/models`.

## Предлагаемый порядок реализации

### Этап 1. Подготовка desktop-каркаса

Сделать:
- добавить `cmd/client/desktop`;
- инициализировать Wails-приложение;
- подключить Vite frontend;
- добавить базовые npm-скрипты;
- зафиксировать команды сборки/запуска в `Makefile`.

Результат:
- окно приложения стартует;
- frontend и Go backend связаны;
- можно вызвать простейший backend method вроде `Ping`/`GetSessionState`.

### Этап 2. Backend bridge поверх ClientCore

Сделать:
- поднять `ClientCore` через `cmd/client/common`;
- создать backend service-слой для auth/records/sync/binary;
- описать DTO и mapping;
- унифицировать обработку ошибок.

Результат:
- весь нужный функционал доступен из React без прямого знания о transport/cache.

### Этап 3. Auth flow

Сделать:
- стартовый экран с `login/register`;
- login screen;
- register screen;
- success modal после регистрации;
- автологин через `OK`, как в TUI;
- восстановление сессии на старте.

Результат:
- пользователь может зайти или зарегистрироваться и сразу попасть в приложение.

### Этап 4. Основной shell приложения

Сделать:
- layout после логина;
- header/status area;
- table списка записей;
- detail-pane;
- фильтр по типу;
- logout.

Результат:
- приложение уже пригодно для `list/get/logout`.

### Этап 5. CRUD для записей

Сделать:
- формы `add/update` для всех типов записей;
- удаление с подтверждением;
- file picker для binary upload;
- save dialog для binary download.

Результат:
- desktop поддерживает полный CRUD-паритет с CLI/TUI.

### Этап 6. Sync и polish

Сделать:
- отдельное действие `sync`;
- статус последней синхронизации;
- loading states;
- optimistic disable на время запросов;
- пустые состояния и понятные ошибки.

Результат:
- desktop-клиент становится рабочим повседневным интерфейсом.

### Этап 7. Документация и проверка сборки

Сделать:
- обновить `README`;
- добавить команды запуска desktop-клиента;
- добавить smoke-проверки и минимальные тесты на mapping/DTO;
- проверить, что CLI/TUI не сломались.

## Изменения в инфраструктуре и зависимостях

Понадобится:
- Go-зависимость на `Wails`;
- frontend toolchain:
  - `npm`
  - `vite`
  - `react`
  - `typescript`
  - `antd`
- генерация Wails bindings между Go и frontend.

Желательно заранее определить:
- нужен ли `wails doctor` как часть setup;
- где хранить frontend lockfile;
- будет ли desktop-сборка входить в CI сразу или отдельным follow-up.

## Тестирование

### Go-уровень

Покрыть тестами:
- DTO mapping;
- auth orchestration backend-слоя;
- бинарные операции без UI;
- ошибки валидации и преобразования payload.

### Frontend-уровень

Минимально проверить:
- auth screens;
- register success modal;
- автологин после register;
- rendering списка записей;
- формы add/update по типам;
- delete confirmation.

### Smoke / manual

Проверить руками:
- запуск без сохранённой сессии;
- восстановление сохранённой сессии;
- login;
- register -> `registered successfully` -> `OK` -> auto-login;
- add/get/update/delete для `login`, `text`, `card`, `binary`;
- download binary на диск;
- sync;
- logout.

## Риски и решения

### 1. Дублирование логики между CLI/TUI/Desktop

Риск:
- часть логики снова уедет во frontend.

Решение:
- максимально использовать `ClientCore`, `cmd/client/common`, `pkg/clientui`;
- новые преобразования класть в shared helper/DTO-mapper, а не в React-компоненты.

### 2. Сложная сериализация payload между Go и TS

Риск:
- неудобно прокидывать `interface`/union payload из Go в frontend.

Решение:
- сразу ввести flat DTO для UI, а не прокидывать `models.Record` как есть.

### 3. Binary UX

Риск:
- бинарные записи ведут себя иначе, чем текстовые.

Решение:
- разделить сценарии `metadata/details` и `file transfer`;
- использовать file chooser/save dialog через Wails runtime API или backend helper.

### 4. Смешение состояния UI и серверного состояния

Риск:
- трудно держать выбранную запись, таблицу и формы в консистентном состоянии.

Решение:
- держать один источник истины в React store на уровне desktop shell;
- после мутаций перечитывать список и детали.

## Definition of Done

Задачу можно считать завершённой, когда:
- есть отдельный desktop-клиент в `cmd/client/desktop`;
- приложение стартует через Wails и подключается к тому же серверу;
- стартовый экран показывает только `login` и `register`;
- после `register` показывается modal `registered successfully` с `OK`;
- `OK` делает auto-login теми же данными;
- после логина доступны `list`, `get`, `add`, `update`, `delete`, `sync`, `logout`;
- бинарные файлы можно загрузить и сохранить;
- logout возвращает на стартовый экран;
- README и команды запуска обновлены.

## Рекомендуемый первый инкремент

Самый безопасный первый кусок реализации:
- поднять каркас `Wails`;
- сделать backend bridge к `ClientCore`;
- реализовать только `session restore`, `login`, `register`, `register success modal`, `logout`;
- после этого переходить к records CRUD.

Так мы быстро проверим главную интеграцию `Wails <-> Go <-> ClientCore`, не размазываясь сразу на всю таблицу и формы.
