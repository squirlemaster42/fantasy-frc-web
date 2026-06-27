package assert

import (
	"context"
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

func (a *assert) printContext(ctx context.Context, msg string, extra ...any) {
	args := []any{"name", a.name}
	args = append(args, extra...)
	_, file, line, ok := runtime.Caller(2)
	if ok {
		args = append(args, "file", file, "line", line)
	}
	for k, v := range a.context {
		args = append(args, k, v)
	}
	log.Fatal(ctx, "assertion failed: "+msg, args...)
}

func (a *assert) NoError(ctx context.Context, err error, msg string) {
	if err != nil {
		a.printContext(ctx, msg, "error", err)
	}
}

func AssertCF(ctx context.Context, predicate bool, msg string) {
	if !predicate {
		args := []any{}
		_, file, line, ok := runtime.Caller(1)
		if ok {
			args = append(args, "file", file, "line", line)
		}
		log.Fatal(ctx, "assertion failed: "+msg, args...)
	}
}

func NoErrorCF(ctx context.Context, err error, msg string) {
	if err != nil {
		args := []any{"error", err}
		_, file, line, ok := runtime.Caller(1)
		if ok {
			args = append(args, "file", file, "line", line)
		}
		log.Fatal(ctx, "assertion failed: "+msg, args...)
	}
}
