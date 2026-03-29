// Package clientcore реализует use-case слой клиентского приложения GophKeeper.
//
// Границы ответственности:
//   - Оркестрирует операции уровня приложения: Login, Register, Logout,
//     ListRecords, GetRecord, SaveRecord, DeleteRecord, SyncNow,
//     UploadBinary, DownloadBinary.
//   - Управляет офлайн-режимом: сохраняет pending-операции при отсутствии связи,
//     отправляет их при синхронизации.
//   - Поддерживает resume upload/download через TransferState из cache.
//   - Не зависит от конкретного UI (CLI, desktop) — только от интерфейсов
//     apiclient.Transport и cache.Store.
//
// CLI и другие клиенты создают ClientCore через конструктор New(),
// передавая выбранную реализацию Transport и Store.
package clientcore
