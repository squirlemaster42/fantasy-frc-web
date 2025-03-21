package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseArgString(t *testing.T) {
    argStr := "-s=\"Test Draft\" -t=test -w"
    argMap, _ := ParseArgString(argStr)
    assert.Equal(t, "Test Draft", argMap["s"])
    assert.Equal(t, "test", argMap["t"])
    _, hasVal := argMap["w"]
    assert.True(t, hasVal)
}

func TestFindNextExpirationTime(t *testing.T) {
    assert.Equal(t, time.Date(2025, time.April, 5, 10, 0, 0, 0, nil), time.Date(2025, time.April, 5, 13, 0, 0, 0, nil))
    assert.Equal(t, time.Date(2025, time.April, 5, 12, 0, 0, 0, nil), time.Date(2025, time.April, 5, 15, 0, 0, 0, nil))
    assert.Equal(t, time.Date(2025, time.April, 5, 11, 0, 0, 0, nil), time.Date(2025, time.April, 5, 14, 0, 0, 0, nil))
    assert.Equal(t, time.Date(2025, time.April, 6, 10, 0, 0, 0, nil), time.Date(2025, time.April, 6, 13, 0, 0, 0, nil))
    assert.Equal(t, time.Date(2025, time.April, 6, 11, 0, 0, 0, nil), time.Date(2025, time.April, 6, 14, 0, 0, 0, nil))
    assert.Equal(t, time.Date(2025, time.April, 7, 10, 0, 0, 0, nil), time.Date(2025, time.April, 7, 13, 0, 0, 0, nil))
    assert.Equal(t, time.Date(2025, time.April, 7, 11, 0, 0, 0, nil), time.Date(2025, time.April, 7, 14, 0, 0, 0, nil))
    assert.Equal(t, time.Date(2025, time.April, 8, 10, 0, 0, 0, nil), time.Date(2025, time.April, 8, 13, 0, 0, 0, nil))
    assert.Equal(t, time.Date(2025, time.April, 8, 11, 0, 0, 0, nil), time.Date(2025, time.April, 8, 14, 0, 0, 0, nil))
    assert.Equal(t, time.Date(2025, time.April, 9, 10, 0, 0, 0, nil), time.Date(2025, time.April, 9, 13, 0, 0, 0, nil))
    assert.Equal(t, time.Date(2025, time.April, 9, 11, 0, 0, 0, nil), time.Date(2025, time.April, 9, 14, 0, 0, 0, nil))
    assert.Equal(t, time.Date(2025, time.April, 10, 10, 0, 0, 0, nil), time.Date(2025, time.April, 10, 13, 0, 0, 0, nil))
    assert.Equal(t, time.Date(2025, time.April, 10, 11, 0, 0, 0, nil), time.Date(2025, time.April, 10, 14, 0, 0, 0, nil))
    assert.Equal(t, time.Date(2025, time.April, 11, 10, 0, 0, 0, nil), time.Date(2025, time.April, 11, 13, 0, 0, 0, nil))
    assert.Equal(t, time.Date(2025, time.April, 11, 11, 0, 0, 0, nil), time.Date(2025, time.April, 11, 14, 0, 0, 0, nil))
    assert.Equal(t, time.Date(2025, time.April, 12, 10, 0, 0, 0, nil), time.Date(2025, time.April, 12, 13, 0, 0, 0, nil))
    assert.Equal(t, time.Date(2025, time.April, 12, 11, 0, 0, 0, nil), time.Date(2025, time.April, 12, 14, 0, 0, 0, nil))
    assert.Equal(t, time.Date(2025, time.April, 13, 10, 0, 0, 0, nil), time.Date(2025, time.April, 13, 13, 0, 0, 0, nil))
    assert.Equal(t, time.Date(2025, time.April, 13, 11, 0, 0, 0, nil), time.Date(2025, time.April, 13, 14, 0, 0, 0, nil))
    assert.Equal(t, time.Date(2025, time.April, 14, 10, 0, 0, 0, nil), time.Date(2025, time.April, 14, 13, 0, 0, 0, nil))
    assert.Equal(t, time.Date(2025, time.April, 14, 11, 0, 0, 0, nil), time.Date(2025, time.April, 14, 14, 0, 0, 0, nil))
    assert.Equal(t, time.Date(2025, time.April, 15, 10, 0, 0, 0, nil), time.Date(2025, time.April, 15, 13, 0, 0, 0, nil))
    assert.Equal(t, time.Date(2025, time.April, 15, 11, 0, 0, 0, nil), time.Date(2025, time.April, 15, 14, 0, 0, 0, nil))
    assert.Equal(t, time.Date(2025, time.April, 16, 10, 0, 0, 0, nil), time.Date(2025, time.April, 16, 13, 0, 0, 0, nil))
    assert.Equal(t, time.Date(2025, time.April, 16, 11, 0, 0, 0, nil), time.Date(2025, time.April, 16, 14, 0, 0, 0, nil))
}
