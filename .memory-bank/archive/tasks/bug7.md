# Название
Починить билд тестирования в github

# Описание
Прикладываю Также этот билд занимал 2 минуты 20 секунд. Не может ли быть такого, что там стоит какой-нибудь тайм-аут на 2 минуты и именно из-за этого падает проект. Потому что локально покрытие тестами хорошее. Также стоит убедиться, что никакие зависимости не отличаются от того, что сейчас имеется локально. Возможно, в этом какая-то причина.


Лог:
```
Run make cover-check
  make cover-check
  shell: /usr/bin/bash -e {0}
  env:
    GOIMPORTS_VERSION: v0.39.0
    GOLANGCI_LINT_VERSION: v2.11.3
    PROTOC_GEN_GO_VERSION: v1.36.11
    PROTOC_GEN_GO_GRPC_VERSION: v1.6.1
go test -v -race -coverprofile=coverage.out -coverpkg="$(go list ./... | grep -Ev '/mocks($|/)|/pbv1($|/)|/proto/v1($|/)|/tests($|/)|/cmd/client($|/)' | tr '\n' ',' | sed 's/,$//')" $(go list ./... | grep -Ev '/mocks($|/)|/pbv1($|/)|/proto/v1($|/)|/tests($|/)|/cmd/client($|/)')
go: downloading github.com/wailsapp/wails/v2 v2.11.0
go: downloading github.com/gdamore/tcell/v2 v2.9.0
go: downloading github.com/rivo/tview v0.42.0
go: downloading github.com/jackc/pgx/v5 v5.7.6
go: downloading github.com/rs/zerolog v1.34.0
go: downloading github.com/gojuno/minimock/v3 v3.4.7
go: downloading golang.org/x/term v0.40.0
go: downloading github.com/go-chi/chi/v5 v5.2.5
go: downloading google.golang.org/grpc v1.79.1
go: downloading github.com/pressly/goose/v3 v3.25.0
go: downloading golang.org/x/time v0.15.0
go: downloading google.golang.org/protobuf v1.36.10
go: downloading github.com/golang-jwt/jwt/v5 v5.3.1
go: downloading golang.org/x/crypto v0.46.0
go: downloading github.com/aws/aws-sdk-go-v2 v1.41.2
go: downloading github.com/aws/aws-sdk-go-v2/service/s3 v1.96.2
go: downloading github.com/aws/smithy-go v1.24.2
go: downloading github.com/gdamore/encoding v1.0.1
go: downloading github.com/lucasb-eyer/go-colorful v1.2.0
go: downloading github.com/mattn/go-runewidth v0.0.16
go: downloading golang.org/x/sys v0.41.0
go: downloading golang.org/x/text v0.32.0
go: downloading github.com/rivo/uniseg v0.4.7
go: downloading github.com/davecgh/go-spew v1.1.1
go: downloading github.com/pmezard/go-difflib v1.0.0
go: downloading github.com/mattn/go-colorable v0.1.13
go: downloading github.com/sethvargo/go-retry v0.3.0
go: downloading go.uber.org/multierr v1.11.0
go: downloading golang.org/x/net v0.48.0
go: downloading google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217
go: downloading github.com/jackc/pgpassfile v1.0.0
go: downloading github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761
go: downloading github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.5
go: downloading github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.18
go: downloading github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.18
go: downloading github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.5
go: downloading github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.10
go: downloading github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.18
go: downloading github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.18
go: downloading github.com/leaanthony/u v1.1.1
go: downloading github.com/leaanthony/slicer v1.6.0
go: downloading github.com/jackc/puddle/v2 v2.2.2
go: downloading github.com/mfridman/interpolate v0.0.2
go: downloading golang.org/x/sync v0.19.0
go: downloading github.com/mattn/go-isatty v0.0.20
go: downloading github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.18
go: downloading github.com/leaanthony/go-ansi-parser v1.6.1
go: downloading github.com/pkg/errors v0.9.1
go: downloading github.com/stretchr/testify v1.11.1
go: downloading gopkg.in/yaml.v3 v3.0.1
...
выполнение тестов
...
FAIL
make: *** [Makefile:16: test] Error 1
Error: Process completed with exit code 2.
```