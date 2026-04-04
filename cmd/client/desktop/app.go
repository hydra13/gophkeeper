package main

import (
	"context"
	"sync"
)

type runtimeContextSetter interface {
	SetRuntimeContext(context.Context)
}

type App struct {
	cleanup func()
	setters []runtimeContextSetter
	once    sync.Once
}

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
