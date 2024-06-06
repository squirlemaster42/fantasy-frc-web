package assert

import (
	"log"
	"log/slog"
)

type Assert struct {
    name string
    context map[string]any
}

func CreateAssertWithContext(name string) *Assert {
    return &Assert{
        name: name,
    }
}

func (a *Assert) addContext(key string, value any) {
    a.context[key] = value
}

func (a *Assert) removeContext(key string) {
    delete(a.context, key)
}

func (a *Assert) runAssert(predicate bool, msg string) {
    if !predicate {
        a.printContext(msg)
    }
}

func (a *Assert) printContext(msg string) {
    for k, v := range a.context {
        slog.Error(a.name, "key", k, "value", v)
    }
    log.Fatal(msg)
}

func (a *Assert) noError(err error, msg string) {
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

