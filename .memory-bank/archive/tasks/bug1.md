# Название
Скрытие паролей в предпросмотре

# Статус: DONE

# Описание
В предпросмотре на странице списка секретов на вебе и на десктопе отображается пароль в открытом виде. Лучше отображать пароль при просмотре по клику на кнопку, как это происходит при выводе пароля.

# Анализ

## Затронутые файлы
- `cmd/client/web/src/features/records/RecordDetailsPane.tsx` — веб-клиент
- `cmd/client/desktop/frontend/src/features/records/RecordDetailsPane.tsx` — десктоп-клиент

## Проблема
В `RecordDetailsPane` чувствительные поля отображались открытым текстом:
- `record.payload.password` (тип login)
- `record.payload.number` (тип card — номер карты)
- `record.payload.cvv` (тип card)

## Решение
Добавлен компонент `SecretField` внутри `RecordDetailsPane.tsx`:
- По умолчанию показывает `••••••••` вместо значения
- Кнопка с иконкой глаза (`EyeOutlined`/`EyeInvisibleOutlined`) переключает видимость
- Кнопка копирования доступна только когда значение видно
- Использует `@ant-design/icons` (уже есть в зависимостях обоих клиентов)

## Что изменено
1. Добавлен `import` для `EyeInvisibleOutlined`, `EyeOutlined` из `@ant-design/icons`
2. Добавлен `import { useState }` из `react`
3. Создан компонент `SecretField` с локальным стейтом видимости
4. Пароль, номер карты и CVV теперь обёрнуты в `<SecretField value={...} />`

## Статистика
- Изменено файлов: 2
- Добавлено строк: ~34 (17 строк на файл — компонент SecretField + замены)
- Pre-existing TS-ошибки (не связаны с багом): несоответствие имён пропов между Workspace.tsx и RecordDetailsPane.tsx в обоих клиентах
