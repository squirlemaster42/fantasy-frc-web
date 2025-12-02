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
    assert.Equal(t, time.Date(2025, time.April, 7, 18, 0, 0, 0, time.Local), GetPickExpirationTime(time.Date(2025, time.April, 7, 17, 0, 0, 0, time.Local)))
    assert.Equal(t, time.Date(2025, time.April, 8, 18, 0, 0, 0, time.Local), GetPickExpirationTime(time.Date(2025, time.April, 7, 22, 0, 0, 0, time.Local)))
    assert.Equal(t, time.Date(2025, time.April, 7, 20, 0, 0, 0, time.Local), GetPickExpirationTime(time.Date(2025, time.April, 7, 19, 0, 0, 0, time.Local)))
    assert.Equal(t, time.Date(2025, time.April, 7, 21, 0, 0, 0, time.Local), GetPickExpirationTime(time.Date(2025, time.April, 7, 20, 0, 0, 0, time.Local)))
    assert.Equal(t, time.Date(2025, time.April, 6, 9, 0, 0, 0, time.Local), GetPickExpirationTime(time.Date(2025, time.April, 6,  8, 0, 0, 0, time.Local)))
    assert.Equal(t, time.Date(2025, time.April, 6, 18, 0, 0, 0, time.Local), GetPickExpirationTime(time.Date(2025, time.April, 6, 17, 0, 0, 0, time.Local)))
    assert.Equal(t, time.Date(2025, time.April, 7, 18, 0, 0, 0, time.Local), GetPickExpirationTime(time.Date(2025, time.April, 6, 22, 0, 0, 0, time.Local)))
    assert.Equal(t, time.Date(2025, time.April, 11, 18, 0, 0, 0, time.Local), GetPickExpirationTime(time.Date(2025, time.April, 11, 14, 0, 0, 0, time.Local)))
    assert.Equal(t, time.Date(2025, time.April, 12, 9, 0, 0, 0, time.Local), GetPickExpirationTime(time.Date(2025, time.April, 11, 23, 0, 0, 0, time.Local)))
}

func TestCompareMatches(t *testing.T) {
    assert.True(t, CompareMatchOrder("2024cur_f1m1", "2024cur_f1m2"))
    assert.False(t, CompareMatchOrder("2024cur_f1m1", "2024cur_qm1"))
    assert.True(t, CompareMatchOrder("2024cur_qm10", "2024cur_qm112"))
    assert.False(t, CompareMatchOrder("2024cur_qm116", "2024cur_qm11"))
    assert.True(t, CompareMatchOrder("2024cur_sf2m1", "2024cur_sf9m1"))
    assert.False(t, CompareMatchOrder("2024cur_f1m2", "2024cur_sf12m1"))
    assert.True(t, CompareMatchOrder("2024cur_qm90", "2024cur_sf12m1"))
    assert.False(t, CompareMatchOrder("2024cur_sf12m1", "2024cur_qm72"))
    assert.True(t, CompareMatchOrder("2024cur_qm71", "2024cur_qm72"))
    assert.True(t, CompareMatchOrder("2024cur_qm7", "2024cur_qm72"))
}

func TestGetMatchLevel(t *testing.T) {
    assert.Equal(t, "f", getMatchLevel("2024cur_f1m2"))
    assert.Equal(t, "qm", getMatchLevel("2024cur_qm1"))
    assert.Equal(t, "qm", getMatchLevel("2024cur_qm112"))
    assert.Equal(t, "qm", getMatchLevel("2024cur_qm11"))
    assert.Equal(t, "sf", getMatchLevel("2024cur_sf9m1"))
    assert.Equal(t, "sf", getMatchLevel("2024cur_sf12m1"))
    assert.Equal(t, "sf", getMatchLevel("2024cur_sf12m1"))
    assert.Equal(t, "qm", getMatchLevel("2025cur_qm72"))
}
