package main

import (
	"context"
	"sync"
)

type runtimeContextSetter interface {
	SetRuntimeContext(context.Context)
}

// App управляет жизненным циклом desktop-клиента.
type App struct {
	cleanup func()
	setters []runtimeContextSetter
	once    sync.Once
}

// NewApp создает Wails-приложение и регистрирует сервисы, которым нужен runtime context.
func NewApp(cleanup func(), setters ...runtimeContextSetter) *App {
	return &App{
		cleanup: cleanup,
		setters: setters,
	}
}

func (a *App) startup(ctx context.Context) {
	for _, setter := range a.setters {
		setter.SetRuntimeContext(ctx)
	}
}

func (a *App) shutdown(context.Context) {
	a.once.Do(func() {
		if a.cleanup != nil {
			a.cleanup()
		}
	})
}
