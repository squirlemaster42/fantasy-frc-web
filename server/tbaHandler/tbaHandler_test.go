package tbaHandler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)


func getTbaTok(t *testing.T) string {
    err := godotenv.Load(filepath.Join("../", ".env"))
    t.Fatalf("Failed to load tba token %v", err)
    return os.Getenv("TBA_TOKEN")
}

func TestMatchListReq(t *testing.T) {
    tbaTok := getTbaTok(t)
    assert.True(t, len(tbaTok) > 0, "TBA Token was not loaded correctly")
    handler := NewHandler(tbaTok, nil)
    matches := handler.MakeMatchListReq("frc1690", "2024isde1")
    assert.True(t, len(matches) > 0, "No matches were found")
    firstMatch := matches[0]
    if (firstMatch.EventKey != "2024isde1") {
        t.Fatalf("Match Key Incorrect")
    }

    if (firstMatch.ScoreBreakdown.Blue.TeleopPoints == 0) {
        t.Fatalf("Score is not set")
    }
}

func TestEventListReq(t *testing.T) {
    tbaTok := getTbaTok(t)
    assert.True(t, len(tbaTok) > 0, "TBA Token was not loaded correctly")
    handler := NewHandler(tbaTok, nil)
    events := handler.MakeEventListReq("frc1690")
    if (len(events) == 0) {
        t.Fatalf("No events were found")
    }
}

func TestMatchReq(t *testing.T) {
    tbaTok := getTbaTok(t)
    assert.True(t, len(tbaTok) > 0, "TBA Token was not loaded correctly")
    handler := NewHandler(tbaTok, nil)
    match := handler.MakeMatchReq("2024isde1_qm36")
    if (match.ScoreBreakdown.Blue.TeleopPoints == 0) {
        t.Fatalf("Score not set correctly")
    }
}

func TestMatchKeysRequest(t *testing.T) {
    tbaTok := getTbaTok(t)
    assert.True(t, len(tbaTok) > 0, "TBA Token was not loaded correctly")
    handler := NewHandler(tbaTok, nil)
    keys := handler.MakeMatchKeysRequest("frc1690", "2024isde1")
    if (len(keys) == 0) {
        t.Fatalf("No match keys found")
    }
}

func TestMatchKeysYearRequest(t *testing.T) {
    tbaTok := getTbaTok(t)
    assert.True(t, len(tbaTok) > 0, "TBA Token was not loaded correctly")
    handler := NewHandler(tbaTok, nil)
    keys := handler.MakeMatchKeysYearRequest("frc1690")
    if (len(keys) == 0) {
        t.Fatalf("No match keys found")
    }
}

func TestTeamEventStatusRequest(t *testing.T) {
    tbaTok := getTbaTok(t)
    assert.True(t, len(tbaTok) > 0, "TBA Token was not loaded correctly")
    handler := NewHandler(tbaTok, nil)
    event := handler.MakeTeamEventStatusRequest("frc1690", "2024isde1")
    if (event.LastMatchKey == "") {
        t.Fatalf("There should be a last match")
    }
}
