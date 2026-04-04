package buildinfo

// Version хранит версию сборки.
var Version = "dev"

// BuildDate хранит дату сборки.
var BuildDate = "unknown"

// Short возвращает краткое представление версии.
func Short() string {
	return Version + " (" + BuildDate + ")"
}
