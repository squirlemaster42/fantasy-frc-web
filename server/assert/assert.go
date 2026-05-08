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

func (a *assert) RunAssert(context context.Context, predicate bool, msg string) {
	if !predicate {
		a.printContext(context, msg)
	}
}

func (a *assert) printContext(context context.Context, msg string) {
	log.Error(context, "assertion failed", "name", a.name)
	_, file, line, ok := runtime.Caller(2)
	if ok {
		log.Error(context, "assertion failed", "line", line, "file", file)
	}
	for k, v := range a.context {
		log.Error(context, "assertion context", "key", k, "value", v)
	}
	stdlog.Fatal(msg)
}

func (a *assert) NoError(context context.Context, err error, msg string) {
	if err != nil {
		log.Error(context, "NoError#error encountered", "error", err)
		a.printContext(context, msg)
	}
}

func AssertCF(context context.Context, predicate bool, msg string) {
	if !predicate {
		_, file, line, ok := runtime.Caller(1)
		if ok {
			log.Error(context, "assertion failed", "line", line, "file", file)
		}
		stdlog.Fatal(msg)
	}
}

func NoErrorCF(context context.Context, err error, msg string) {
	if err != nil {
		_, file, line, ok := runtime.Caller(1)
		if ok {
			log.Error(context, "assertion failed", "line", line, "file", file)
		}
		log.Error(context, "NoError#error encountered", "error", err)
		stdlog.Fatal(msg)
	}
}
