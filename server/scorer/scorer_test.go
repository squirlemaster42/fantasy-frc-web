package scorer

import (
	"context"
	"os"
	"path/filepath"
	"server/swagger"
	"server/tbaHandler"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func getTbaTok(t *testing.T) string {
    err := godotenv.Load(filepath.Join("../", ".env"))
    if err != nil {
        t.Fatal(err)
    }
    return os.Getenv("TBA_TOKEN")
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

    s := NewScorer(nil, nil, nil, nil)
    sorted := s.sortMatchesByPlayOrder(t.Context(), unsorted)

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

func TestScoreMatches(t *testing.T) {
    //We should not need a tba handler or database
    tbaHandler := tbaHandler.NewHandler(getTbaTok(t), nil)
    scorer := NewScorer(tbaHandler, nil, nil, nil)
    match, _ := tbaHandler.MakeMatchReq(t.Context(), "2026casnv_qm1")
    scoredMatch, _ := scorer.scoreMatch(t.Context(), match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 0, scoredMatch.RedScore)
    assert.Equal(t, 5, scoredMatch.BlueScore)

    match, _ = tbaHandler.MakeMatchReq(t.Context(), "2026casnv_qm20")
    scoredMatch, _ = scorer.scoreMatch(t.Context(), match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 0, scoredMatch.RedScore)
    assert.Equal(t, 3, scoredMatch.BlueScore)

    match, _ = tbaHandler.MakeMatchReq(t.Context(), "2026casnv_qm38")
    scoredMatch, _ = scorer.scoreMatch(t.Context(), match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 4, scoredMatch.RedScore)
    assert.Equal(t, 1, scoredMatch.BlueScore)

    match, _ = tbaHandler.MakeMatchReq(t.Context(), "2026casnv_qm52")
    scoredMatch, _ = scorer.scoreMatch(t.Context(), match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 0, scoredMatch.RedScore)
    assert.Equal(t, 5, scoredMatch.BlueScore)

    match, _ = tbaHandler.MakeMatchReq(t.Context(), "2026casnv_qm74")
    scoredMatch, _ = scorer.scoreMatch(t.Context(), match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 0, scoredMatch.RedScore)
    assert.Equal(t, 4, scoredMatch.BlueScore)

    match, _ = tbaHandler.MakeMatchReq(t.Context(), "2026casnv_qm24")
    scoredMatch, _ = scorer.scoreMatch(t.Context(), match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 1, scoredMatch.RedScore)
    assert.Equal(t, 5, scoredMatch.BlueScore)

    match, _ = tbaHandler.MakeMatchReq(t.Context(), "2026mawne_qm40")
    scoredMatch, _ = scorer.scoreMatch(t.Context(), match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 6, scoredMatch.RedScore)
    assert.Equal(t, 1, scoredMatch.BlueScore)

    match, _ = tbaHandler.MakeMatchReq(t.Context(), "2026casnv_sf4m1")
    scoredMatch, _ = scorer.scoreMatch(t.Context(), match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 15, scoredMatch.RedScore)
    assert.Equal(t, 0, scoredMatch.BlueScore)

    match, _ = tbaHandler.MakeMatchReq(t.Context(), "2026casnv_sf6m1")
    scoredMatch, _ = scorer.scoreMatch(t.Context(), match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 9, scoredMatch.RedScore)
    assert.Equal(t, 0, scoredMatch.BlueScore)

    match, _ = tbaHandler.MakeMatchReq(t.Context(), "2026casnv_f1m1")
    scoredMatch, _ = scorer.scoreMatch(t.Context(), match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 18, scoredMatch.RedScore)
    assert.Equal(t, 0, scoredMatch.BlueScore)

    match, _ = tbaHandler.MakeMatchReq(t.Context(), "2024cmptx_sf2m1")
    scoredMatch, _ = scorer.scoreMatch(t.Context(), match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 0, scoredMatch.RedScore)
    assert.Equal(t, 15, scoredMatch.BlueScore)

    match, _ = tbaHandler.MakeMatchReq(t.Context(), "2024cmptx_sf12m1")
    scoredMatch, _ = scorer.scoreMatch(t.Context(), match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 9, scoredMatch.RedScore)
    assert.Equal(t, 0, scoredMatch.BlueScore)

    match, _ = tbaHandler.MakeMatchReq(t.Context(), "2024cmptx_f1m1")
    scoredMatch, _ = scorer.scoreMatch(t.Context(), match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 0, scoredMatch.RedScore)
    assert.Equal(t, 18, scoredMatch.BlueScore)
}

type mockTBAHandler struct{}

func (m *mockTBAHandler) MakeEventListReq(ctx context.Context, teamId string) ([]string, error) { return nil, nil }
func (m *mockTBAHandler) MakeMatchReq(ctx context.Context, matchId string) (swagger.Match, error) { return swagger.Match{}, nil }
func (m *mockTBAHandler) MakeEventMatchKeysRequest(ctx context.Context, eventId string) ([]string, error) { return nil, nil }
func (m *mockTBAHandler) MakeTeamsAtEventRequest(ctx context.Context, eventId string) ([]swagger.Team, error) { return nil, nil }
func (m *mockTBAHandler) MakeEliminationAllianceRequest(ctx context.Context, eventId string) ([]swagger.EliminationAlliance, error) { return nil, nil }
func (m *mockTBAHandler) MakeTeamAvatarRequest(ctx context.Context, teamId string) (string, error) { return "", nil }

func TestScorer_RunScorer_WaitReturnsAfterCancel(t *testing.T) {
    scorer := NewScorer(&mockTBAHandler{}, nil, nil, nil)

    ctx, cancel := context.WithCancel(context.Background())
    done := scorer.RunScorer(ctx)

    // Give the goroutine a moment to enter the loop
    time.Sleep(10 * time.Millisecond)

    cancel()

    select {
    case <-done:
        // success
    case <-time.After(100 * time.Millisecond):
        t.Fatal("RunScorer wait did not return after context cancellation")
    }
}

func newSyntheticMatch(
	t *testing.T,
	key, compLevel string,
	setNumber, matchNumber int32,
	winningAlliance, eventKey string,
	postResultTime int64,
	redTeams, blueTeams []string,
	redDq, redSurrogate, blueDq, blueSurrogate []string,
	scoreBreakdown *swagger.OneOfMatchScoreBreakdown,
) swagger.Match {
	t.Helper()
	return swagger.Match{
		Key:             key,
		CompLevel:       compLevel,
		SetNumber:       setNumber,
		MatchNumber:     matchNumber,
		WinningAlliance: winningAlliance,
		EventKey:        eventKey,
		PostResultTime:  postResultTime,
		Alliances: &swagger.MatchSimpleAlliances{
			Red: &swagger.MatchAlliance{
				Score:             0,
				TeamKeys:          redTeams,
				SurrogateTeamKeys: redSurrogate,
				DqTeamKeys:        redDq,
			},
			Blue: &swagger.MatchAlliance{
				Score:             0,
				TeamKeys:          blueTeams,
				SurrogateTeamKeys: blueSurrogate,
				DqTeamKeys:        blueDq,
			},
		},
		ScoreBreakdown: scoreBreakdown,
	}
}

func newSyntheticScoreBreakdown(red, blue *swagger.MatchScoreBreakdown2026Alliance) *swagger.OneOfMatchScoreBreakdown {
	return &swagger.OneOfMatchScoreBreakdown{
		MatchScoreBreakdown2026: swagger.MatchScoreBreakdown2026{
			Red:  red,
			Blue: blue,
		},
	}
}

func TestScoreMatchSynthetic(t *testing.T) {
	scorer := NewScorer(&mockTBAHandler{}, nil, nil, nil)
	ctx := t.Context()

	t.Run("unplayed match rescore false returns not scored", func(t *testing.T) {
		match := newSyntheticMatch(t, "2026arc_qm1", "qm", 1, 1, "red", "2026arc", 0,
			[]string{"frc1", "frc2", "frc3"}, []string{"frc4", "frc5", "frc6"},
			nil, nil, nil, nil, nil)
		scored, shouldUpdate := scorer.scoreMatch(ctx, match, false)
		assert.False(t, scored.Played)
		assert.False(t, shouldUpdate)
		assert.Equal(t, 0, scored.RedScore)
		assert.Equal(t, 0, scored.BlueScore)
	})

	t.Run("unplayed match rescore true scores anyway", func(t *testing.T) {
		match := newSyntheticMatch(t, "2026arc_qm2", "qm", 1, 2, "blue", "2026arc", 0,
			[]string{"frc1", "frc2", "frc3"}, []string{"frc4", "frc5", "frc6"},
			nil, nil, nil, nil, nil)
		scored, shouldUpdate := scorer.scoreMatch(ctx, match, true)
		assert.False(t, scored.Played)
		assert.True(t, shouldUpdate)
		assert.Equal(t, 0, scored.RedScore)
		assert.Equal(t, QualWinPoints(), scored.BlueScore)
	})

	t.Run("qual red wins", func(t *testing.T) {
		match := newSyntheticMatch(t, "2026arc_qm3", "qm", 1, 3, "red", "2026arc", 1,
			[]string{"frc1", "frc2", "frc3"}, []string{"frc4", "frc5", "frc6"},
			nil, nil, nil, nil, nil)
		scored, shouldUpdate := scorer.scoreMatch(ctx, match, false)
		assert.True(t, scored.Played)
		assert.True(t, shouldUpdate)
		assert.Equal(t, QualWinPoints(), scored.RedScore)
		assert.Equal(t, 0, scored.BlueScore)
	})

	t.Run("qual blue wins", func(t *testing.T) {
		match := newSyntheticMatch(t, "2026arc_qm4", "qm", 1, 4, "blue", "2026arc", 1,
			[]string{"frc1", "frc2", "frc3"}, []string{"frc4", "frc5", "frc6"},
			nil, nil, nil, nil, nil)
		scored, _ := scorer.scoreMatch(ctx, match, false)
		assert.Equal(t, 0, scored.RedScore)
		assert.Equal(t, QualWinPoints(), scored.BlueScore)
	})

	t.Run("qual bonuses for red and blue", func(t *testing.T) {
		breakdown := newSyntheticScoreBreakdown(
			&swagger.MatchScoreBreakdown2026Alliance{
				EnergizedAchieved:    true,
				SuperchargedAchieved: true,
				TraversalAchieved:    true,
			},
			&swagger.MatchScoreBreakdown2026Alliance{
				EnergizedAchieved: true,
			},
		)
		match := newSyntheticMatch(t, "2026arc_qm5", "qm", 1, 5, "red", "2026arc", 1,
			[]string{"frc1", "frc2", "frc3"}, []string{"frc4", "frc5", "frc6"},
			nil, nil, nil, nil, breakdown)
		scored, _ := scorer.scoreMatch(ctx, match, false)
		expectedRed := QualWinPoints() + EnergizedBonusPoints() + SuperchargedBonusPoints() + TraversalBonusPoints()
		expectedBlue := EnergizedBonusPoints()
		assert.Equal(t, expectedRed, scored.RedScore)
		assert.Equal(t, expectedBlue, scored.BlueScore)
	})

	t.Run("playoff finals red wins", func(t *testing.T) {
		match := newSyntheticMatch(t, "2026arc_f1m1", "f", 1, 1, "red", "2026arc", 1,
			[]string{"frc1", "frc2", "frc3"}, []string{"frc4", "frc5", "frc6"},
			nil, nil, nil, nil, nil)
		scored, _ := scorer.scoreMatch(ctx, match, false)
		assert.Equal(t, PlayoffFinalsPoints(), scored.RedScore)
		assert.Equal(t, 0, scored.BlueScore)
	})

	t.Run("playoff semifinals upper bracket blue wins", func(t *testing.T) {
		match := newSyntheticMatch(t, "2026arc_sf1m1", "sf", 1, 1, "blue", "2026arc", 1,
			[]string{"frc1", "frc2", "frc3"}, []string{"frc4", "frc5", "frc6"},
			nil, nil, nil, nil, nil)
		scored, _ := scorer.scoreMatch(ctx, match, false)
		assert.Equal(t, 0, scored.RedScore)
		assert.Equal(t, PlayoffUpperBracketPoints(), scored.BlueScore)
	})

	t.Run("playoff semifinals lower bracket red wins", func(t *testing.T) {
		match := newSyntheticMatch(t, "2026arc_sf5m1", "sf", 5, 1, "red", "2026arc", 1,
			[]string{"frc1", "frc2", "frc3"}, []string{"frc4", "frc5", "frc6"},
			nil, nil, nil, nil, nil)
		scored, _ := scorer.scoreMatch(ctx, match, false)
		assert.Equal(t, PlayoffLowerBracketPoints(), scored.RedScore)
		assert.Equal(t, 0, scored.BlueScore)
	})

	t.Run("einstein upper bracket red wins multiplied", func(t *testing.T) {
		match := newSyntheticMatch(t, "2026cmptx_sf1m1", "sf", 1, 1, "red", "2026cmptx", 1,
			[]string{"frc1", "frc2", "frc3"}, []string{"frc4", "frc5", "frc6"},
			nil, nil, nil, nil, nil)
		scored, _ := scorer.scoreMatch(ctx, match, false)
		expected := PlayoffUpperBracketPoints() * EinsteinMultiplier()
		assert.Equal(t, expected, scored.RedScore)
		assert.Equal(t, 0, scored.BlueScore)
	})

	t.Run("dq and surrogate teams included in dqed teams", func(t *testing.T) {
		match := newSyntheticMatch(t, "2026arc_qm6", "qm", 1, 6, "red", "2026arc", 1,
			[]string{"frc1", "frc2", "frc3"}, []string{"frc4", "frc5", "frc6"},
			[]string{"frc2"}, []string{"frc3"},
			[]string{"frc5"}, []string{"frc6"}, nil)
		scored, _ := scorer.scoreMatch(ctx, match, false)
		assert.ElementsMatch(t, []string{"frc2", "frc3", "frc5", "frc6"}, scored.DqedTeams)
	})

	t.Run("alliance selection score maps picks to base score times multiplier", func(t *testing.T) {
		alliance := swagger.EliminationAlliance{
			Name:  "Alliance 1",
			Picks: []string{"frc254", "frc1678", "frc118", "frc2056"},
		}
		scores := scorer.GetAllianceSelectionScore(ctx, alliance)
		assert.Equal(t, 32*AlliancePickMultiplier(), scores["frc254"])
		assert.Equal(t, 31*AlliancePickMultiplier(), scores["frc1678"])
		assert.Equal(t, 9*AlliancePickMultiplier(), scores["frc118"])
		assert.Equal(t, 8*AlliancePickMultiplier(), scores["frc2056"])
	})
}

func TestGetAllianceSelectionScores (t *testing.T) {
    tbaHandler := tbaHandler.NewHandler(getTbaTok(t), nil)
    alliances, _ := tbaHandler.MakeEliminationAllianceRequest(t.Context(), "2025mawor")
    scorer := NewScorer(tbaHandler, nil, nil, nil)
    allianceOneScores := scorer.GetAllianceSelectionScore(t.Context(), alliances[0])
    assert.EqualValues(t, 32 * 2, allianceOneScores["frc190"])
    assert.EqualValues(t, 31 * 2, allianceOneScores["frc1768"])
    assert.EqualValues(t, 9 * 2, allianceOneScores["frc3182"])
    allianceTwoScores := scorer.GetAllianceSelectionScore(t.Context(), alliances[1])
    assert.EqualValues(t, 30 * 2, allianceTwoScores["frc125"])
    assert.EqualValues(t, 29 * 2, allianceTwoScores["frc88"])
    assert.EqualValues(t, 10 * 2, allianceTwoScores["frc8626"])
    allianceThreeScores := scorer.GetAllianceSelectionScore(t.Context(), alliances[2])
    assert.EqualValues(t, 28 * 2, allianceThreeScores["frc1153"])
    assert.EqualValues(t, 27 * 2, allianceThreeScores["frc230"])
    assert.EqualValues(t, 11 * 2, allianceThreeScores["frc2079"])
    allianceFourScores := scorer.GetAllianceSelectionScore(t.Context(), alliances[3])
    assert.EqualValues(t, 26 * 2, allianceFourScores["frc2370"])
    assert.EqualValues(t, 25 * 2, allianceFourScores["frc1100"])
    assert.EqualValues(t, 12 * 2, allianceFourScores["frc1757"])
    allianceFiveScores := scorer.GetAllianceSelectionScore(t.Context(), alliances[4])
    assert.EqualValues(t, 24 * 2, allianceFiveScores["frc1277"])
    assert.EqualValues(t, 23 * 2, allianceFiveScores["frc2067"])
    assert.EqualValues(t, 13 * 2, allianceFiveScores["frc126"])
    allianceSixScores := scorer.GetAllianceSelectionScore(t.Context(), alliances[5])
    assert.EqualValues(t, 22 * 2, allianceSixScores["frc5459"])
    assert.EqualValues(t, 21 * 2, allianceSixScores["frc1699"])
    assert.EqualValues(t, 14 * 2, allianceSixScores["frc1740"])
    allianceSevenScores := scorer.GetAllianceSelectionScore(t.Context(), alliances[6])
    assert.EqualValues(t, 20 * 2, allianceSevenScores["frc5000"])
    assert.EqualValues(t, 19 * 2, allianceSevenScores["frc1735"])
    assert.EqualValues(t, 15 * 2, allianceSevenScores["frc1119"])
    allianceEightScores := scorer.GetAllianceSelectionScore(t.Context(), alliances[7])
    assert.EqualValues(t, 18 * 2, allianceEightScores["frc7153"])
    assert.EqualValues(t, 17 * 2, allianceEightScores["frc5422"])
    assert.EqualValues(t, 16 * 2, allianceEightScores["frc9644"])
}
