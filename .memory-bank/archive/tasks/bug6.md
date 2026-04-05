Упал билд проверки литинга в гитхаб. Посмотри на этот лог ошибки и поправь:
```
run golangci-lint
  Running [/home/runner/golangci-lint-2.11.3-linux-amd64/golangci-lint config path] in [/home/runner/work/gophkeeper/gophkeeper] ...
  Running [/home/runner/golangci-lint-2.11.3-linux-amd64/golangci-lint config verify] in [/home/runner/work/gophkeeper/gophkeeper] ...
  Running [/home/runner/golangci-lint-2.11.3-linux-amd64/golangci-lint run] in [/home/runner/work/gophkeeper/gophkeeper] ...
  Error: cmd/client/cli/commands/auth.go:33:14: Error return value of `fmt.Fprintln` is not checked (errcheck)
  	fmt.Fprintln(r.deps.Stdout, "registered successfully")
  	            ^
  Error: cmd/client/cli/commands/auth.go:60:14: Error return value of `fmt.Fprintln` is not checked (errcheck)
  	fmt.Fprintln(r.deps.Stdout, "logged in successfully")
  	            ^
  Error: cmd/client/cli/commands/records.go:40:13: Error return value of `fmt.Fprintf` is not checked (errcheck)
  	fmt.Fprintf(r.deps.Stdout, "%-8s %-10s %-30s %-10s\t%s\n", "ID", "TYPE", "NAME", "REVISION", "METADATA")
  	           ^
  Error: cmd/client/cli/commands/records.go:81:15: Error return value of `fmt.Fprintf` is not checked (errcheck)
  			fmt.Fprintf(r.deps.Stdout, "Data size: %d bytes\n", len(data))
  			           ^
  Error: cmd/client/cli/commands/records.go:89:14: Error return value of `fmt.Fprintf` is not checked (errcheck)
  		fmt.Fprintf(r.deps.Stdout, "saved %d bytes to %s\n", len(data), outputPath)
  		           ^
  Error: internal/api/auth_login_v1_post/handler_test.go:138:25: Error return value of `resp.Body.Close` is not checked (errcheck)
  			defer resp.Body.Close()
  			                     ^
  Error: internal/api/auth_logout_v1_post/handler_test.go:107:25: Error return value of `resp.Body.Close` is not checked (errcheck)
  			defer resp.Body.Close()
  			                     ^
  Error: internal/api/auth_register_v1_post/handler_test.go:126:25: Error return value of `resp.Body.Close` is not checked (errcheck)
  			defer resp.Body.Close()
  			                     ^
  Error: internal/api/sync_pull_v1_post/handler_test.go:224:32: Error return value of `(*encoding/json.Decoder).Decode` is not checked (errcheck)
  	json.NewDecoder(w.Body).Decode(&resp)
  	                              ^
  Error: internal/api/sync_push_v1_post/handler.go:147:27: Error return value of `(*encoding/json.Encoder).Encode` is not checked (errcheck)
  	json.NewEncoder(w).Encode(resp)
  	                         ^
  Error: internal/api/sync_push_v1_post/handler_test.go:119:32: Error return value of `(*encoding/json.Decoder).Decode` is not checked (errcheck)
  	json.NewDecoder(w.Body).Decode(&resp)
  	                              ^
  Error: internal/api/uploads_by_id_chunks_v1_get/handler.go:57:27: Error return value of `(*encoding/json.Encoder).Encode` is not checked (errcheck)
  	json.NewEncoder(w).Encode(resp)
  	                         ^
  Error: internal/api/uploads_by_id_chunks_v1_post/handler.go:102:27: Error return value of `(*encoding/json.Encoder).Encode` is not checked (errcheck)
  	json.NewEncoder(w).Encode(resp)
  	                         ^
  Error: internal/api/uploads_by_id_chunks_v1_post/handler_test.go:78:32: Error return value of `(*encoding/json.Decoder).Decode` is not checked (errcheck)
  	json.NewDecoder(w.Body).Decode(&resp)
  	                              ^
  Error: internal/config/loader_test.go:207:16: Error return value of `os.Chdir` is not checked (errcheck)
  	defer os.Chdir(origWd)
  	              ^
  Error: internal/config/loader_test.go:391:16: Error return value of `os.Chdir` is not checked (errcheck)
  	defer os.Chdir(origWd)
  	              ^
  Error: internal/middlewares/compression.go:33:18: Error return value of `gz.Close` is not checked (errcheck)
  			defer gz.Close()
  			              ^
  Error: internal/middlewares/compression_test.go:34:16: Error return value of `gr.Close` is not checked (errcheck)
  	defer gr.Close()
  	              ^
  Error: internal/middlewares/compression_test.go:118:16: Error return value of `gr.Close` is not checked (errcheck)
  	defer gr.Close()
  	              ^
  Error: internal/repositories/database/repository.go:221:18: Error return value of `rows.Close` is not checked (errcheck)
  	defer rows.Close()
  	                ^
  Error: internal/repositories/database/repository.go:256:18: Error return value of `rows.Close` is not checked (errcheck)
  	defer rows.Close()
  	                ^
  Error: internal/repositories/database/repository.go:382:18: Error return value of `rows.Close` is not checked (errcheck)
  	defer rows.Close()
  	                ^
  Error: internal/repositories/database/repository.go:743:24: Error return value of `chunksRows.Close` is not checked (errcheck)
  	defer chunksRows.Close()
  	                      ^
  Error: internal/repositories/database/repository.go:793:24: Error return value of `chunksRows.Close` is not checked (errcheck)
  	defer chunksRows.Close()
  	                      ^
  Error: internal/repositories/database/repository_unit_test.go:200:16: Error return value of `db.Close` is not checked (errcheck)
  	defer db.Close()
  	              ^
  Error: internal/repositories/database/repository_unit_test.go:224:16: Error return value of `db.Close` is not checked (errcheck)
  	defer db.Close()
  	              ^
  Error: pkg/apiclient/grpc/client_test.go:751:14: Error return value of `client.Close` is not checked (errcheck)
  	client.Close()
  	            ^
  Error: pkg/apiclient/grpc/transport_test.go:1138:17: Error return value of `lis.Close` is not checked (errcheck)
  	defer lis.Close()
  	               ^
  Error: pkg/apiclient/grpc/transport_test.go:1144:14: Error return value of `srv.Serve` is not checked (errcheck)
  	go srv.Serve(lis)
  	            ^
  Error: pkg/apiclient/grpc/transport_test.go:1156:20: Error return value of `client.Close` is not checked (errcheck)
  	defer client.Close()
  	                  ^
  Error: pkg/apiclient/grpc/transport_test.go:1170:17: Error return value of `lis.Close` is not checked (errcheck)
  	defer lis.Close()
  	               ^
  Error: pkg/cache/json_test.go:502:26: Error return value of `(github.com/hydra13/gophkeeper/pkg/cache.PendingQueue).Enqueue` is not checked (errcheck)
  	store1.Pending().Enqueue(PendingOp{
  	                        ^
  Error: pkg/cache/json_test.go:506:26: Error return value of `(github.com/hydra13/gophkeeper/pkg/cache.PendingQueue).Enqueue` is not checked (errcheck)
  	store1.Pending().Enqueue(PendingOp{
  	                        ^
  Error: pkg/cache/json_test.go:510:26: Error return value of `(github.com/hydra13/gophkeeper/pkg/cache.PendingQueue).Enqueue` is not checked (errcheck)
  	store1.Pending().Enqueue(PendingOp{
  	                        ^
  Error: pkg/cache/memory_test.go:107:12: Error return value of `pq.Enqueue` is not checked (errcheck)
  	pq.Enqueue(PendingOp{RecordID: 3, Operation: OperationDelete})
  	          ^
  Error: pkg/cache/memory_test.go:145:9: Error return value of `ts.Save` is not checked (errcheck)
  	ts.Save(tr)
  	       ^
  Error: pkg/cache/memory_test.go:152:9: Error return value of `ts.Save` is not checked (errcheck)
  	ts.Save(tr)
  	       ^
  Error: pkg/cache/memory_test.go:161:9: Error return value of `ts.Save` is not checked (errcheck)
  	ts.Save(Transfer{ID: 2, Status: TransferStatusActive})
  	       ^
  Error: pkg/cache/memory_test.go:226:19: Error return value of `(github.com/hydra13/gophkeeper/pkg/cache.AuthStore).Set` is not checked (errcheck)
  	store1.Auth().Set(AuthData{
  	                 ^
  Error: pkg/cache/memory_test.go:230:31: Error return value of `(github.com/hydra13/gophkeeper/pkg/cache.SyncState).SetLastRevision` is not checked (errcheck)
  	store1.Sync().SetLastRevision(10)
  	                             ^
  Error: pkg/clientcore/core.go:51:20: Error return value of `(github.com/hydra13/gophkeeper/pkg/cache.AuthStore).Set` is not checked (errcheck)
  	c.store.Auth().Set(cache.AuthData{
  	                  ^
  Error: pkg/clientcore/core.go:73:20: Error return value of `(github.com/hydra13/gophkeeper/pkg/cache.AuthStore).Set` is not checked (errcheck)
  	c.store.Auth().Set(cache.AuthData{
  	                  ^
  Error: pkg/clientcore/core.go:94:32: Error return value of `(github.com/hydra13/gophkeeper/pkg/cache.SyncState).SetLastRevision` is not checked (errcheck)
  	c.store.Sync().SetLastRevision(0)
  	                              ^
  Error: pkg/clientcore/core.go:147:33: Error return value of `(github.com/hydra13/gophkeeper/pkg/cache.SyncState).SetLastRevision` is not checked (errcheck)
  		c.store.Sync().SetLastRevision(maxRevision)
  		                              ^
  Error: pkg/clientcore/core.go:247:17: Error return value of `rec.SoftDelete` is not checked (errcheck)
  		rec.SoftDelete()
  		              ^
  Error: pkg/clientcore/core.go:398:27: Error return value of `(github.com/hydra13/gophkeeper/pkg/cache.TransferState).Save` is not checked (errcheck)
  		c.store.Transfers().Save(cache.Transfer{
  		                        ^
  Error: pkg/clientcore/core.go:470:27: Error return value of `(github.com/hydra13/gophkeeper/pkg/cache.TransferState).Save` is not checked (errcheck)
  		c.store.Transfers().Save(*transfer)
  		                        ^
  Error: pkg/clientcore/core.go:483:27: Error return value of `(github.com/hydra13/gophkeeper/pkg/cache.TransferState).Save` is not checked (errcheck)
  		c.store.Transfers().Save(*transfer)
  		                        ^
  Error: pkg/clientui/records.go:85:12: Error return value of `fmt.Fprint` is not checked (errcheck)
  	fmt.Fprint(w, FormatRecord(rec))
  	          ^
  Error: pkg/clientui/records.go:89:14: Error return value of `fmt.Fprintln` is not checked (errcheck)
  	fmt.Fprintln(w, FormatRecordShort(rec))
  	            ^
  Error: internal/repositories/database/repository.go:213:3: ineffectual assignment to argIdx (ineffassign)
  		argIdx++
  		^
  Error: internal/services/uploads/service_test.go:654:13: ineffectual assignment to total (ineffassign)
  	confirmed, total, status, err := svc.ConfirmChunk(downloadID, 0)
  	           ^
  Error: pkg/clientcore/core.go:426:3: ineffectual assignment to startChunk (ineffassign)
  		startChunk = transfer.CompletedIdx + 1
  		^
  Error: internal/api/records_by_id_binary_v1_get/handler.go:61:3: QF1002: could use tagged switch on err (staticcheck)
  		switch {
  		^
  Error: internal/middlewares/grpc_test.go:378:55: SA1029: should not use built-in type string as key for value; define your own type to avoid collisions (staticcheck)
  	customCtx := context.WithValue(context.Background(), "test-key", "test-value")
  	                                                     ^
  Error: internal/repositories/database/repository.go:208:12: S1039: unnecessary use of fmt.Sprintf (staticcheck)
  		query += fmt.Sprintf(" AND deleted_at IS NULL")
  		         ^
  Error: cmd/client/cli/bridge.go:37:6: func defaultCacheDir is unused (unused)
  func defaultCacheDir() string {
       ^
  Error: cmd/client/cli/bridge.go:49:6: func hostname is unused (unused)
  func hostname() string {
       ^
  Error: internal/app/server_test.go:106:6: type mockSyncService is unused (unused)
  type mockSyncService struct {
       ^
  Error: internal/app/server_test.go:114:27: func (*mockSyncService).Push is unused (unused)
  func (m *mockSyncService) Push(userID int64, deviceID string, changes []models.PendingChange) ([]models.RecordRevision, []models.SyncConflict, error) {
                            ^
  Error: internal/app/server_test.go:119:27: func (*mockSyncService).Pull is unused (unused)
  func (m *mockSyncService) Pull(userID int64, deviceID string, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error) {
                            ^
  Error: internal/app/server_test.go:123:27: func (*mockSyncService).GetConflicts is unused (unused)
  func (m *mockSyncService) GetConflicts(userID int64) ([]models.SyncConflict, error) {
                            ^
  Error: internal/app/server_test.go:127:27: func (*mockSyncService).ResolveConflict is unused (unused)
  func (m *mockSyncService) ResolveConflict(userID int64, conflictID int64, resolution string) (*models.Record, error) {
                            ^
  Error: internal/services/uploads/service_test.go:206:6: func containsChunk is unused (unused)
  func containsChunk(chunks []models.Chunk, chunkIndex int64) bool {
       ^
  64 issues:
  * errcheck: 50
  * ineffassign: 3
  * staticcheck: 3
  * unused: 8
  
  Error: issues found
  Ran golangci-lint in 38956ms

```