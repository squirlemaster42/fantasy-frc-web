package assert

import (
	"log"
	"log/slog"
	"runtime"
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
    _, file, line, ok := runtime.Caller(2)
    if ok {
        log.Printf("Failed at: line %d in %s", line, file)
    }
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
        _, file, line, ok := runtime.Caller(1)
        log.Printf("Failed at: line %d in %s", line, file)
        if ok {
            log.Printf("Failed at: %d in %s", line, file)
        }
        log.Fatal(msg)
    }
}

func NoErrorCF(err error, msg string) {
    if err != nil {
        _, file, line, ok := runtime.Caller(1)
        if ok {
            log.Printf("Failed at: line %d in %s", line, file)
        }
        slog.Error("NoError#error encountered", "error", err)
        log.Fatal(msg)
    }
}

