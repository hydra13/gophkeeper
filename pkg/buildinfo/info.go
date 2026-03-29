package buildinfo

// Version — версия бинарника. Задаётся через -ldflags "-X github.com/hydra13/gophkeeper/pkg/buildinfo.Version=...".
// По умолчанию "dev".
var Version = "dev"

// BuildDate — дата сборки бинарника. Задаётся через -ldflags "-X github.com/hydra13/gophkeeper/pkg/buildinfo.BuildDate=...".
// По умолчанию "unknown".
var BuildDate = "unknown"

// Short возвращает краткую строку версии и даты сборки.
func Short() string {
	return Version + " (" + BuildDate + ")"
}
