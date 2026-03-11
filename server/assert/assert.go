package assert

import (
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

func (a *assert) RunAssert(predicate bool, msg string) {
	if !predicate {
		a.printContext(msg)
	}
}

func (a *assert) printContext(msg string) {
	log.ErrorNoContext("assertion failed", "name", a.name)
	_, file, line, ok := runtime.Caller(2)
	if ok {
		log.ErrorNoContext("assertion failed", "line", line, "file", file)
	}
	for k, v := range a.context {
		log.ErrorNoContext("assertion context", "key", k, "value", v)
	}
	stdlog.Fatal(msg)
}

func (a *assert) NoError(err error, msg string) {
	if err != nil {
		log.ErrorNoContext("NoError#error encountered", "error", err)
		a.printContext(msg)
	}
}

func AssertCF(predicate bool, msg string) {
	if !predicate {
		_, file, line, ok := runtime.Caller(1)
		if ok {
			log.ErrorNoContext("assertion failed", "line", line, "file", file)
		}
		stdlog.Fatal(msg)
	}
}

func NoErrorCF(err error, msg string) {
	if err != nil {
		_, file, line, ok := runtime.Caller(1)
		if ok {
			log.ErrorNoContext("assertion failed", "line", line, "file", file)
		}
		log.ErrorNoContext("NoError#error encountered", "error", err)
		stdlog.Fatal(msg)
	}
}
