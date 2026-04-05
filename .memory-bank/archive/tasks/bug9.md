1) Продолжают падать интеграционные тесты. Нужно поправить:
```
Run make test-integration
go test -p 1 -tags=integration -count=1 ./internal/jobs/reencrypt ./internal/repositories/database ./internal/services/sync ./internal/services/uploads ./internal/storage
go: downloading github.com/jackc/pgx/v5 v5.7.6
go: downloading github.com/stretchr/testify v1.11.1
go: downloading github.com/aws/aws-sdk-go-v2 v1.41.2
go: downloading github.com/aws/aws-sdk-go-v2/service/s3 v1.96.2
go: downloading github.com/aws/smithy-go v1.24.2
go: downloading github.com/gojuno/minimock/v3 v3.4.7
go: downloading github.com/pressly/goose/v3 v3.25.0
go: downloading github.com/jackc/pgpassfile v1.0.0
go: downloading github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761
go: downloading golang.org/x/crypto v0.46.0
go: downloading golang.org/x/text v0.32.0
go: downloading github.com/davecgh/go-spew v1.1.1
go: downloading github.com/pmezard/go-difflib v1.0.0
go: downloading github.com/jackc/puddle/v2 v2.2.2
go: downloading github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.5
go: downloading github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.18
go: downloading github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.18
go: downloading github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.5
go: downloading github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.10
go: downloading github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.18
go: downloading github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.18
go: downloading gopkg.in/yaml.v3 v3.0.1
go: downloading golang.org/x/sync v0.19.0
go: downloading github.com/sethvargo/go-retry v0.3.0
go: downloading go.uber.org/multierr v1.11.0
go: downloading github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.18
go: downloading github.com/mfridman/interpolate v0.0.2
ok  	github.com/hydra13/gophkeeper/internal/jobs/reencrypt	0.205s
ok  	github.com/hydra13/gophkeeper/internal/repositories/database	0.293s
2026/04/04 19:26:44 goose: no migrations to run. current version: 3
2026/04/04 19:26:44 goose: no migrations to run. current version: 3
--- FAIL: TestSyncConflictOnConcurrentUpdate (0.04s)
    service_integration_test.go:96: 
        	Error Trace:	/home/runner/work/gophkeeper/gophkeeper/internal/services/sync/service_integration_test.go:96
        	Error:      	"[]" should have 1 item(s), but has 0
        	Test:       	TestSyncConflictOnConcurrentUpdate
2026/04/04 19:26:44 goose: no migrations to run. current version: 3
2026/04/04 19:26:44 goose: no migrations to run. current version: 3
2026/04/04 19:26:44 goose: no migrations to run. current version: 3
--- FAIL: TestSyncResolveConflictLocal (0.04s)
    service_integration_test.go:265: 
        	Error Trace:	/home/runner/work/gophkeeper/gophkeeper/internal/services/sync/service_integration_test.go:265
        	Error:      	Received unexpected error:
        	            	record not found
        	Test:       	TestSyncResolveConflictLocal
2026/04/04 19:26:44 goose: no migrations to run. current version: 3
--- FAIL: TestSyncResolveConflictServer (0.04s)
    service_integration_test.go:337: 
        	Error Trace:	/home/runner/work/gophkeeper/gophkeeper/internal/services/sync/service_integration_test.go:337
        	Error:      	Received unexpected error:
        	            	record not found
        	Test:       	TestSyncResolveConflictServer
2026/04/04 19:26:44 goose: no migrations to run. current version: 3
2026/04/04 19:26:44 goose: no migrations to run. current version: 3
--- FAIL: TestSyncStaleDeleteAfterUpdateReturnsConflict (0.04s)
    service_integration_test.go:521: 
        	Error Trace:	/home/runner/work/gophkeeper/gophkeeper/internal/services/sync/service_integration_test.go:521
        	Error:      	"[]" should have 1 item(s), but has 0
        	Test:       	TestSyncStaleDeleteAfterUpdateReturnsConflict
2026/04/04 19:26:44 goose: no migrations to run. current version: 3
2026/04/04 19:26:44 goose: no migrations to run. current version: 3
--- FAIL: TestSyncDeleteConflictReturnsBothVersions (0.04s)
    service_integration_test.go:664: 
        	Error Trace:	/home/runner/work/gophkeeper/gophkeeper/internal/services/sync/service_integration_test.go:664
        	Error:      	Not equal: 
        	            	expected: "server-version"
        	            	actual  : "original"
        	            	
        	            	Diff:
        	            	--- Expected
        	            	+++ Actual
        	            	@@ -1 +1 @@
        	            	-server-version
        	            	+original
        	Test:       	TestSyncDeleteConflictReturnsBothVersions
2026/04/04 19:26:44 goose: no migrations to run. current version: 3
--- FAIL: TestSyncUpdateConflictReturnsBothVersions (0.04s)
    service_integration_test.go:738: 
        	Error Trace:	/home/runner/work/gophkeeper/gophkeeper/internal/services/sync/service_integration_test.go:738
        	Error:      	Not equal: 
        	            	expected: "server-version"
        	            	actual  : "original"
        	            	
        	            	Diff:
        	            	--- Expected
        	            	+++ Actual
        	            	@@ -1 +1 @@
        	            	-server-version
        	            	+original
        	Test:       	TestSyncUpdateConflictReturnsBothVersions
FAIL
FAIL	github.com/hydra13/gophkeeper/internal/services/sync	0.479s
ok  	github.com/hydra13/gophkeeper/internal/services/uploads	0.674s
ok  	github.com/hydra13/gophkeeper/internal/storage	0.013s
FAIL
make: *** [Makefile:30: test-integration] Error 1
Error: Process completed with exit code 2.
```