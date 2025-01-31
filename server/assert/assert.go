package assert

import (
	"log"
	"log/slog"
)

type assert struct {
    name string
    context map[string]any
}

func CreateAssertWithContext(name string) *assert {
    return &assert{
        name: name,
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
    slog.Error(a.name)
    for k, v := range a.context {
        slog.Error("key: ", k, "value: ", v)
    }
    log.Fatal(msg)
}

func (a *assert) NoError(err error, msg string) {
    if err != nil {
        slog.Error("NoError#error encountered", "error", err)
        a.printContext(msg);
    }
}

func AssertCF(predicate bool, msg string) {
    if !predicate {
        log.Fatal(msg)
    }
}

func NoErrorCF(err error, msg string) {
    if err != nil {
        slog.Error("NoError#error encountered", "error", err)
        log.Fatal(msg)
    }
}

