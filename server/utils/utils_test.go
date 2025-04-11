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
    assert.Equal(t, time.Date(2025, time.April, 7, 20, 0, 0, 0, time.Local), GetPickExpirationTime(time.Date(2025, time.April, 7, 17, 0, 0, 0, time.Local)))
    assert.Equal(t, time.Date(2025, time.April, 8, 20, 0, 0, 0, time.Local), GetPickExpirationTime(time.Date(2025, time.April, 7, 22, 0, 0, 0, time.Local)))
    assert.Equal(t, time.Date(2025, time.April, 7, 22, 0, 0, 0, time.Local), GetPickExpirationTime(time.Date(2025, time.April, 7, 19, 0, 0, 0, time.Local)))
    assert.Equal(t, time.Date(2025, time.April, 8, 18, 0, 0, 0, time.Local), GetPickExpirationTime(time.Date(2025, time.April, 7, 20, 0, 0, 0, time.Local)))
    assert.Equal(t, time.Date(2025, time.April, 6, 11, 0, 0, 0, time.Local), GetPickExpirationTime(time.Date(2025, time.April, 6,  8, 0, 0, 0, time.Local)))
    assert.Equal(t, time.Date(2025, time.April, 6, 20, 0, 0, 0, time.Local), GetPickExpirationTime(time.Date(2025, time.April, 6, 17, 0, 0, 0, time.Local)))
    assert.Equal(t, time.Date(2025, time.April, 7, 20, 0, 0, 0, time.Local), GetPickExpirationTime(time.Date(2025, time.April, 6, 22, 0, 0, 0, time.Local)))
    assert.Equal(t, time.Date(2025, time.April, 11, 20, 0, 0, 0, time.Local), GetPickExpirationTime(time.Date(2025, time.April, 11, 14, 0, 0, 0, time.Local)))
    assert.Equal(t, time.Date(2025, time.April, 12, 11, 0, 0, 0, time.Local), GetPickExpirationTime(time.Date(2025, time.April, 11, 23, 0, 0, 0, time.Local)))
}
