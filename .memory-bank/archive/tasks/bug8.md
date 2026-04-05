так не пойдет. давай исправим:
1) всегда для моков используй minimock, который должен генерировать моки через go:generate (хороший пример - сервис [sync](internal/services/sync) )
2) файл с юнит-тестами должен наименоваться просто <имя проверяемого файла>_test.go (не  <имя проверяемого файла>_unit_test.go)
3) каждый модуль, будь это сервис или хендлер, должет быть самодостаточным и использовать только интерфейсы, определенные в нем, а также общие модели из [models](internal/models) (конечно могут быть исключения по моделям - но это лучше уточнять у меня)
4) билд github "unit" падает с ошибкой, надо поправить. лог:
```
--- FAIL: TestUnmarshalConflictRecord_DeletedAt (0.00s)
    repository_unit_test.go:1017: 
        	Error Trace:	/home/runner/work/gophkeeper/gophkeeper/internal/repositories/database/repository_unit_test.go:1017
        	Error:      	Not equal: 
        	            	expected: time.Date(2026, time.April, 4, 14, 6, 12, 0, time.Local)
        	            	actual  : time.Date(2026, time.April, 4, 14, 6, 12, 0, time.UTC)
        	            	
        	            	Diff:
        	Test:       	TestUnmarshalConflictRecord_DeletedAt
FAIL
FAIL	github.com/hydra13/gophkeeper/internal/repositories/database	0.020s
```
5) integration тоже с ошибкой падает:
```
2026/04/04 15:05:25 goose: successfully migrated database to version: 3
--- FAIL: TestJobReencrypt (0.35s)
    job_integration_test.go:63: 
        	Error Trace:	/home/runner/work/gophkeeper/gophkeeper/internal/jobs/reencrypt/job_integration_test.go:63
        	Error:      	Received unexpected error:
        	            	record not found
        	Test:       	TestJobReencrypt
FAIL
FAIL	github.com/hydra13/gophkeeper/internal/jobs/reencrypt	0.359s
--- FAIL: TestSyncPushPullHappyPath (0.10s)
    service_integration_test.go:44: 
        	Error Trace:	/home/runner/work/gophkeeper/gophkeeper/internal/services/sync/service_integration_test.go:44
        	Error:      	Received unexpected error:
        	            	create record: cipher: message authentication failed
        	Test:       	TestSyncPushPullHappyPath
--- FAIL: TestSyncConflictOnConcurrentUpdate (0.06s)
    service_integration_test.go:75: 
        	Error Trace:	/home/runner/work/gophkeeper/gophkeeper/internal/services/sync/service_integration_test.go:75
        	Error:      	Received unexpected error:
        	            	create record: cipher: message authentication failed
        	Test:       	TestSyncConflictOnConcurrentUpdate
--- FAIL: TestSyncSoftDelete (0.09s)
    service_integration_test.go:140: 
        	Error Trace:	/home/runner/work/gophkeeper/gophkeeper/internal/services/sync/service_integration_test.go:140
        	Error:      	Received unexpected error:
        	            	create record: cipher: message authentication failed
        	Test:       	TestSyncSoftDelete
--- FAIL: TestSyncIncrementalPull (0.06s)
    service_integration_test.go:188: 
        	Error Trace:	/home/runner/work/gophkeeper/gophkeeper/internal/services/sync/service_integration_test.go:188
        	Error:      	Received unexpected error:
        	            	create record: cipher: message authentication failed
        	Test:       	TestSyncIncrementalPull
--- FAIL: TestSyncResolveConflictLocal (0.09s)
    service_integration_test.go:222: 
        	Error Trace:	/home/runner/work/gophkeeper/gophkeeper/internal/services/sync/service_integration_test.go:222
        	Error:      	Received unexpected error:
        	            	create record: cipher: message authentication failed
        	Test:       	TestSyncResolveConflictLocal
--- FAIL: TestSyncResolveConflictServer (0.06s)
    service_integration_test.go:294: 
        	Error Trace:	/home/runner/work/gophkeeper/gophkeeper/internal/services/sync/service_integration_test.go:294
        	Error:      	Received unexpected error:
        	            	create record: cipher: message authentication failed
        	Test:       	TestSyncResolveConflictServer
2026/04/04 15:05:32 goose: no migrations to run. current version: 3
--- FAIL: TestSyncRestoreAfterSoftDelete (0.10s)
    service_integration_test.go:361: 
        	Error Trace:	/home/runner/work/gophkeeper/gophkeeper/internal/services/sync/service_integration_test.go:361
        	Error:      	Received unexpected error:
        	            	create record: cipher: message authentication failed
        	Test:       	TestSyncRestoreAfterSoftDelete
2026/04/04 15:05:32 goose: no migrations to run. current version: 3
--- FAIL: TestSyncStaleDeleteAfterUpdateReturnsConflict (0.06s)
    service_integration_test.go:504: 
        	Error Trace:	/home/runner/work/gophkeeper/gophkeeper/internal/services/sync/service_integration_test.go:504
        	Error:      	Received unexpected error:
        	            	create record: cipher: message authentication failed
        	Test:       	TestSyncStaleDeleteAfterUpdateReturnsConflict
2026/04/04 15:05:32 goose: no migrations to run. current version: 3
--- FAIL: TestSyncDeleteThenRestoreViaUpdate (0.10s)
    service_integration_test.go:566: 
        	Error Trace:	/home/runner/work/gophkeeper/gophkeeper/internal/services/sync/service_integration_test.go:566
        	Error:      	Received unexpected error:
        	            	create record: cipher: message authentication failed
        	Test:       	TestSyncDeleteThenRestoreViaUpdate
2026/04/04 15:05:32 goose: no migrations to run. current version: 3
--- FAIL: TestSyncDeleteConflictReturnsBothVersions (0.06s)
    service_integration_test.go:632: 
        	Error Trace:	/home/runner/work/gophkeeper/gophkeeper/internal/services/sync/service_integration_test.go:632
        	Error:      	Received unexpected error:
        	            	create record: cipher: message authentication failed
        	Test:       	TestSyncDeleteConflictReturnsBothVersions
2026/04/04 15:05:32 goose: no migrations to run. current version: 3
--- FAIL: TestSyncUpdateConflictReturnsBothVersions (0.09s)
    service_integration_test.go:697: 
        	Error Trace:	/home/runner/work/gophkeeper/gophkeeper/internal/services/sync/service_integration_test.go:697
        	Error:      	Received unexpected error:
        	            	create record: cipher: message authentication failed
        	Test:       	TestSyncUpdateConflictReturnsBothVersions
FAIL
FAIL	github.com/hydra13/gophkeeper/internal/services/sync	0.886s
2026/04/04 15:05:31 goose: no migrations to run. current version: 3
--- FAIL: TestIntegration_UploadHappyPath (0.16s)
    service_integration_test.go:151: 
        	Error Trace:	/home/runner/work/gophkeeper/gophkeeper/internal/services/uploads/service_integration_test.go:151
        	Error:      	Received unexpected error:
        	            	get upload session: upload session not found
        	Test:       	TestIntegration_UploadHappyPath
2026/04/04 15:05:31 goose: no migrations to run. current version: 3
--- FAIL: TestIntegration_DownloadAfterUpload (0.15s)
    service_integration_test.go:189: 
        	Error Trace:	/home/runner/work/gophkeeper/gophkeeper/internal/services/uploads/service_integration_test.go:189
        	Error:      	Received unexpected error:
        	            	get upload session: upload session not found
        	Test:       	TestIntegration_DownloadAfterUpload
2026/04/04 15:05:32 goose: no migrations to run. current version: 3
--- FAIL: TestIntegration_ResumeUpload (0.16s)
    service_integration_test.go:219: 
        	Error Trace:	/home/runner/work/gophkeeper/gophkeeper/internal/services/uploads/service_integration_test.go:219
        	Error:      	Received unexpected error:
        	            	get upload session: upload session not found
        	Test:       	TestIntegration_ResumeUpload
2026/04/04 15:05:32 goose: no migrations to run. current version: 3
--- FAIL: TestIntegration_DownloadSessionFlow (0.15s)
    service_integration_test.go:261: 
        	Error Trace:	/home/runner/work/gophkeeper/gophkeeper/internal/services/uploads/service_integration_test.go:261
        	Error:      	Received unexpected error:
        	            	get upload session: upload session not found
        	Test:       	TestIntegration_DownloadSessionFlow
2026/04/04 15:05:32 goose: no migrations to run. current version: 3
--- FAIL: TestIntegration_DownloadResume (0.16s)
    service_integration_test.go:308: 
        	Error Trace:	/home/runner/work/gophkeeper/gophkeeper/internal/services/uploads/service_integration_test.go:308
        	Error:      	Received unexpected error:
        	            	get upload session: upload session not found
        	Test:       	TestIntegration_DownloadResume
2026/04/04 15:05:32 goose: no migrations to run. current version: 3
2026/04/04 15:05:32 goose: no migrations to run. current version: 3
2026/04/04 15:05:32 goose: no migrations to run. current version: 3
2026/04/04 15:05:32 goose: no migrations to run. current version: 3
2026/04/04 15:05:32 goose: no migrations to run. current version: 3
2026/04/04 15:05:32 goose: no migrations to run. current version: 3
2026/04/04 15:05:32 goose: no migrations to run. current version: 3
2026/04/04 15:05:32 goose: no migrations to run. current version: 3
FAIL
FAIL	github.com/hydra13/gophkeeper/internal/services/uploads	1.360s
ok  	github.com/hydra13/gophkeeper/internal/storage	0.022s
FAIL
make: *** [Makefile:25: test-integration] Error 1
```
6) coverage также упал, также не смог успеть за 2 минуты (2m 17s), лог содержить более 2600 строк (скорее всего весь вывод тестом, чего для покрытия тестами не обязательно знать) - надо поправить. последние 4 строки из лога:
```
ok  	github.com/hydra13/gophkeeper/internal/storage	1.070s	coverage: 2.9% of statements in github.com/hydra13/gophkeeper/cmd/server, github.com/hydra13/gophkeeper/internal/api/auth_login_v1_post, github.com/hydra13/gophkeeper/internal/api/auth_logout_v1_post, github.com/hydra13/gophkeeper/internal/api/auth_refresh_v1_post, github.com/hydra13/gophkeeper/internal/api/auth_register_v1_post, github.com/hydra13/gophkeeper/internal/api/health_v1_get, github.com/hydra13/gophkeeper/internal/api/records_by_id_binary_v1_get, github.com/hydra13/gophkeeper/internal/api/records_by_id_v1_delete, github.com/hydra13/gophkeeper/internal/api/records_by_id_v1_get, github.com/hydra13/gophkeeper/internal/api/records_by_id_v1_put, github.com/hydra13/gophkeeper/internal/api/records_common, github.com/hydra13/gophkeeper/internal/api/records_v1_get, github.com/hydra13/gophkeeper/internal/api/records_v1_post, github.com/hydra13/gophkeeper/internal/api/sync_pull_v1_post, github.com/hydra13/gophkeeper/internal/api/sync_push_v1_post, github.com/hydra13/gophkeeper/internal/api/uploads_by_id_chunks_v1_get, github.com/hydra13/gophkeeper/internal/api/uploads_by_id_chunks_v1_post, github.com/hydra13/gophkeeper/internal/api/uploads_by_id_v1_get, github.com/hydra13/gophkeeper/internal/api/uploads_v1_post, github.com/hydra13/gophkeeper/internal/app, github.com/hydra13/gophkeeper/internal/config, github.com/hydra13/gophkeeper/internal/jobs/reencrypt, github.com/hydra13/gophkeeper/internal/middlewares, github.com/hydra13/gophkeeper/internal/migrations, github.com/hydra13/gophkeeper/internal/models, github.com/hydra13/gophkeeper/internal/repositories, github.com/hydra13/gophkeeper/internal/repositories/database, github.com/hydra13/gophkeeper/internal/repositories/file, github.com/hydra13/gophkeeper/internal/rpc, github.com/hydra13/gophkeeper/internal/services/auth, github.com/hydra13/gophkeeper/internal/services/crypto, github.com/hydra13/gophkeeper/internal/services/data, github.com/hydra13/gophkeeper/internal/services/keys, github.com/hydra13/gophkeeper/internal/services/passwords, github.com/hydra13/gophkeeper/internal/services/records, github.com/hydra13/gophkeeper/internal/services/sync, github.com/hydra13/gophkeeper/internal/services/uploads, github.com/hydra13/gophkeeper/internal/services/users, github.com/hydra13/gophkeeper/internal/services/validation, github.com/hydra13/gophkeeper/internal/storage
FAIL
make: *** [Makefile:20: test] Error 1
Error: Process completed with exit code 2.
```