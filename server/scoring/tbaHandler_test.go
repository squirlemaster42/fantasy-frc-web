package scoring

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestMatchListReq(t *testing.T) {
    godotenv.Load()
    tbaTok := os.Getenv("TBA_TOKEN")
    handler := NewHandler(tbaTok)
    matches := handler.makeMatchListReq("frc1690", "2024isde1")
    firstMatch := matches[0]
    if (firstMatch.EventKey != "2024isde1") {
        t.Fatalf("Match Key Incorrect")
    }

    if (firstMatch.ScoreBreakdown.Blue.TeleopPoints == 0) {
        t.Fatalf("Score is not set")
    }
}

func TestEventListReq(t *testing.T) {
    godotenv.Load()
    tbaTok := os.Getenv("TBA_TOKEN")
    handler := NewHandler(tbaTok)
    events := handler.makeEventListReq("frc1690")
    if (len(events) == 0) {
        t.Fatalf("No events were found")
    }
}

func TestMatchReq(t *testing.T) {
    godotenv.Load()
    tbaTok := os.Getenv("TBA_TOKEN")
    handler := NewHandler(tbaTok)
    match := handler.makeMatchReq("2024isde1_qm39")
    if (match.ScoreBreakdown.Blue.TeleopPoints == 0) {
        t.Fatalf("Score not set correctly")
    }
}

func TestMatchKeysRequest(t *testing.T) {
    godotenv.Load()
    tbaTok := os.Getenv("TBA_TOKEN")
    handler := NewHandler(tbaTok)
    keys := handler.makeMatchKeysRequest("frc1690", "2024isde1")
    if (len(keys) == 0) {
        t.Fatalf("No match keys found")
    }
}

func TestMatchKeysYearRequest(t *testing.T) {
    godotenv.Load()
    tbaTok := os.Getenv("TBA_TOKEN")
    handler := NewHandler(tbaTok)
    keys := handler.makeMatchKeysYearRequest("frc1690")
    if (len(keys) == 0) {
        t.Fatalf("No match keys found")
    }
}

func TestTeamEventStatusRequest(t *testing.T) {
    godotenv.Load()
    tbaTok := os.Getenv("TBA_TOKEN")
    handler := NewHandler(tbaTok)
    event := handler.makeTeamEventStatusRequest("frc1690", "2024isde1")
    if (event.LastMatchKey == "") {
        t.Fatalf("There should be a last match")
    }
}
