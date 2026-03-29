// Package apiclient отвечает только за транспортный слой общения с сервером GophKeeper.
//
// Границы ответственности:
//   - Определяет интерфейс Transport — транспортно-независимый контракт для Auth, Records,
//     Sync, Upload/Download операций.
//   - Содержит типы данных транспортного уровня (PullResult, PushResult, UploadStatus и т.д.).
//   - Реализации Transport (gRPC, HTTP REST) живут в подпакетах (grpc, http).
//
// Пакет НЕ содержит бизнес-логики, управления кешем или офлайн-режимом.
// Клиентский use-case слой (clientcore) работает только через интерфейс Transport.
package apiclient
