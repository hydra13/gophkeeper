# Task 5. HTTP auth endpoints в `api/`

## Цель
Подготовить endpoint-first HTTP-контракты и каркас обработчиков для auth-сценариев: `register`, `login`, `refresh`, `logout`.

## Описание
HTTP-слой по плану не является gateway над gRPC, а реализуется как самостоятельный transport. Для auth нужно создать отдельные endpoint-пакеты и зафиксировать request/response DTO, mapping ошибок и тестовый каркас, согласованный с доменом и gRPC-контрактами.

## Последовательность шагов
1. Создать пакеты `api/auth_register_v1_post`, `api/auth_login_v1_post`, `api/auth_refresh_v1_post`, `api/auth_logout_v1_post`.
2. В каждом пакете подготовить `handler.go`, `handler_test.go` и каталог `mocks/`.
3. Описать request/response DTO и правила валидации входных данных.
4. Зафиксировать HTTP status codes и mapping бизнес-ошибок на transport-ответы.
5. Подготовить единый способ dependency injection для endpoint-first пакетов без общего монолитного handler-слоя.
6. Добавить unit-тесты на happy-path, ошибки валидации и неавторизованный доступ.

## Критерии приемки
- [ ] Для всех auth-ручек существуют отдельные пакеты в `api/`.
- [ ] В каждом endpoint-пакете есть `handler.go`, `handler_test.go` и `mocks/`.
- [ ] DTO и mapping ошибок согласованы с доменом и gRPC auth-контрактами.
- [ ] Пакеты можно реализовывать независимо без скрытых общих зависимостей.
- [ ] Есть базовые unit-тесты на успешные и ошибочные auth-сценарии.
