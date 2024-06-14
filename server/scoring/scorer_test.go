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


func TestGetMatchLevel(t *testing.T) {
    assert.Equal(t, "f", getMatchLevel("2024cur_f1m2"))
    assert.Equal(t, "qm", getMatchLevel("2024cur_qm1"))
    assert.Equal(t, "qm", getMatchLevel("2024cur_qm112"))
    assert.Equal(t, "qm", getMatchLevel("2024cur_qm11"))
    assert.Equal(t, "sf",  getMatchLevel("2024cur_sf9m1"))
    assert.Equal(t, "sf", getMatchLevel("2024cur_sf12m1"))
    assert.Equal(t, "sf", getMatchLevel("2024cur_sf12m1"))
    assert.Equal(t, "qm", getMatchLevel("2025cur_qm72"))
}

func TestSortMatchOrder(t *testing.T) {
    unsorted := []string{
        "2024cur_f1m1",
        "2024cur_qf1m1",
        "2024cur_qm1",
        "2024cur_qm100",
        "2024cur_sf1m1",
        "2024cur_sf12m1",
        "2024cur_f1m2",
        "2024cur_qm52",
    }

    sorted := sortMatchesByPlayOrder(unsorted)

    standard := []string{
        "2024cur_qm1",
        "2024cur_qm52",
        "2024cur_qm100",
        "2024cur_qf1m1",
        "2024cur_sf1m1",
        "2024cur_sf12m1",
        "2024cur_f1m1",
        "2024cur_f1m2",
    }

    assert.True(t, len(sorted) == len(standard), "Sorted array is not the correct length")

    for i, match := range standard {
        assert.Equal(t, match, sorted[i])
    }
}
