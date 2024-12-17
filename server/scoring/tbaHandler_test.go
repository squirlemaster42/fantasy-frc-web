package scoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchListReq(t *testing.T) {
    tbaTok := getTbaTok()
    assert.True(t, len(tbaTok) > 0, "TBA Token was not loaded correctly")
    handler := NewHandler(tbaTok)
    matches := handler.makeMatchListReq("frc1690", "2024isde1")
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
    tbaTok := getTbaTok()
    assert.True(t, len(tbaTok) > 0, "TBA Token was not loaded correctly")
    handler := NewHandler(tbaTok)
    events := handler.makeEventListReq("frc1690")
    if (len(events) == 0) {
        t.Fatalf("No events were found")
    }
}

func TestMatchReq(t *testing.T) {
    tbaTok := getTbaTok()
    assert.True(t, len(tbaTok) > 0, "TBA Token was not loaded correctly")
    handler := NewHandler(tbaTok)
    match := handler.makeMatchReq("2024isde1_qm36")
    if (match.ScoreBreakdown.Blue.TeleopPoints == 0) {
        t.Fatalf("Score not set correctly")
    }
}

func TestMatchKeysRequest(t *testing.T) {
    tbaTok := getTbaTok()
    assert.True(t, len(tbaTok) > 0, "TBA Token was not loaded correctly")
    handler := NewHandler(tbaTok)
    keys := handler.makeMatchKeysRequest("frc1690", "2024isde1")
    if (len(keys) == 0) {
        t.Fatalf("No match keys found")
    }
}

func TestMatchKeysYearRequest(t *testing.T) {
    tbaTok := getTbaTok()
    assert.True(t, len(tbaTok) > 0, "TBA Token was not loaded correctly")
    handler := NewHandler(tbaTok)
    keys := handler.makeMatchKeysYearRequest("frc1690")
    if (len(keys) == 0) {
        t.Fatalf("No match keys found")
    }
}

func TestTeamEventStatusRequest(t *testing.T) {
    tbaTok := getTbaTok()
    assert.True(t, len(tbaTok) > 0, "TBA Token was not loaded correctly")
    handler := NewHandler(tbaTok)
    event := handler.makeTeamEventStatusRequest("frc1690", "2024isde1")
    if (event.LastMatchKey == "") {
        t.Fatalf("There should be a last match")
    }
}