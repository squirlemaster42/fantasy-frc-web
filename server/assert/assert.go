package assert

import (
	"context"
	stdlog "log"
	"runtime"
	"server/log"
)

type assert struct {
	name    string
	context map[string]any
}

func CreateAssertWithContext(name string) *assert {
	return &assert{
		name:    name,
		context: make(map[string]any),
	}
}

func (a *assert) AddContext(key string, value any) {
	a.context[key] = value
}

func (a *assert) RemoveContext(key string) {
	delete(a.context, key)
}

func (a *assert) RunAssert(ctx context.Context, predicate bool, msg string) {
	if !predicate {
		a.printContext(ctx, msg)
	}
}

func (a *assert) printContext(ctx context.Context, msg string) {
	log.Error(ctx, "assertion failed", "name", a.name)
	_, file, line, ok := runtime.Caller(2)
	if ok {
		log.Error(ctx, "assertion failed", "line", line, "file", file)
	}
	for k, v := range a.context {
		log.Error(ctx, "assertion context", "key", k, "value", v)
	}
	stdlog.Fatal(msg)
}

func (a *assert) NoError(ctx context.Context, err error, msg string) {
	if err != nil {
		log.Error(ctx, "NoError#error encountered", "error", err)
		a.printContext(ctx, msg)
	}
}

func AssertCF(ctx context.Context, predicate bool, msg string) {
	if !predicate {
		_, file, line, ok := runtime.Caller(1)
		if ok {
			log.Error(ctx, "assertion failed", "line", line, "file", file)
		}
		stdlog.Fatal(msg)
	}
}

func NoErrorCF(ctx context.Context, err error, msg string) {
	if err != nil {
		_, file, line, ok := runtime.Caller(1)
		if ok {
			log.Error(ctx, "assertion failed", "line", line, "file", file)
		}
		log.Error(ctx, "NoError#error encountered", "error", err)
		stdlog.Fatal(msg)
	}
}
