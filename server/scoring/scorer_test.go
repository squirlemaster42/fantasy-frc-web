package scoring

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func getTbaTok() string {
    godotenv.Load(filepath.Join("../", ".env"))
    return os.Getenv("TBA_TOKEN")
}

func TestCompareMatches(t *testing.T) {
    assert.True(t, compareMatchOrder("2024cur_f1m1", "2024cur_f1m2"))
    assert.False(t, compareMatchOrder("2024cur_f1m1", "2024cur_qm1"))
    assert.True(t, compareMatchOrder("2024cur_qm10", "2024cur_qm112"))
    assert.False(t, compareMatchOrder("2024cur_qm116", "2024cur_qm11"))
    assert.True(t, compareMatchOrder("2024cur_sf2m1", "2024cur_sf9m1"))
    assert.False(t, compareMatchOrder("2024cur_f1m2", "2024cur_sf12m1"))
    assert.True(t, compareMatchOrder("2024cur_qm90", "2024cur_sf12m1"))
    assert.False(t, compareMatchOrder("2024cur_sf12m1", "2024cur_qm72"))
}
