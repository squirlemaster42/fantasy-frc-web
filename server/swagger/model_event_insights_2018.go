/*
 * The Blue Alliance API v3
 *
 * # Overview    Information and statistics about FIRST Robotics Competition teams and events.   # Authentication   All endpoints require an Auth Key to be passed in the header `X-TBA-Auth-Key`. If you do not have an auth key yet, you can obtain one from your [Account Page](/account).
 *
 * API version: 3.9.13
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package swagger

// Insights for FIRST Power Up qualification and elimination matches.
type EventInsights2018 struct {
	// An array with three values, number of times auto quest was completed, number of opportunities to complete the auto quest, and percentage.
	AutoQuestAchieved []float32 `json:"auto_quest_achieved"`
	// Average number of boost power up scored (out of 3).
	AverageBoostPlayed float32 `json:"average_boost_played"`
	// Average endgame points.
	AverageEndgamePoints float32 `json:"average_endgame_points"`
	// Average number of force power up scored (out of 3).
	AverageForcePlayed float32 `json:"average_force_played"`
	// Average foul score.
	AverageFoulScore float32 `json:"average_foul_score"`
	// Average points scored during auto.
	AveragePointsAuto float32 `json:"average_points_auto"`
	// Average points scored during teleop.
	AveragePointsTeleop float32 `json:"average_points_teleop"`
	// Average mobility points scored during auto.
	AverageRunPointsAuto float32 `json:"average_run_points_auto"`
	// Average scale ownership points scored.
	AverageScaleOwnershipPoints float32 `json:"average_scale_ownership_points"`
	// Average scale ownership points scored during auto.
	AverageScaleOwnershipPointsAuto float32 `json:"average_scale_ownership_points_auto"`
	// Average scale ownership points scored during teleop.
	AverageScaleOwnershipPointsTeleop float32 `json:"average_scale_ownership_points_teleop"`
	// Average score.
	AverageScore float32 `json:"average_score"`
	// Average switch ownership points scored.
	AverageSwitchOwnershipPoints float32 `json:"average_switch_ownership_points"`
	// Average switch ownership points scored during auto.
	AverageSwitchOwnershipPointsAuto float32 `json:"average_switch_ownership_points_auto"`
	// Average switch ownership points scored during teleop.
	AverageSwitchOwnershipPointsTeleop float32 `json:"average_switch_ownership_points_teleop"`
	// Average value points scored.
	AverageVaultPoints float32 `json:"average_vault_points"`
	// Average margin of victory.
	AverageWinMargin float32 `json:"average_win_margin"`
	// Average winning score.
	AverageWinScore float32 `json:"average_win_score"`
	// An array with three values, number of times a boost power up was played, number of opportunities to play a boost power up, and percentage.
	BoostPlayedCounts []float32 `json:"boost_played_counts"`
	// An array with three values, number of times a climb occurred, number of opportunities to climb, and percentage.
	ClimbCounts []float32 `json:"climb_counts"`
	// An array with three values, number of times an alliance faced the boss, number of opportunities to face the boss, and percentage.
	FaceTheBossAchieved []float32 `json:"face_the_boss_achieved"`
	// An array with three values, number of times a force power up was played, number of opportunities to play a force power up, and percentage.
	ForcePlayedCounts []float32 `json:"force_played_counts"`
	// An array with three values, high score, match key from the match with the high score, and the name of the match
	HighScore []string `json:"high_score"`
	// An array with three values, number of times a levitate power up was played, number of opportunities to play a levitate power up, and percentage.
	LevitatePlayedCounts []float32 `json:"levitate_played_counts"`
	// An array with three values, number of times a team scored mobility points in auto, number of opportunities to score mobility points in auto, and percentage.
	RunCountsAuto []float32 `json:"run_counts_auto"`
	// Average scale neutral percentage.
	ScaleNeutralPercentage float32 `json:"scale_neutral_percentage"`
	// Average scale neutral percentage during auto.
	ScaleNeutralPercentageAuto float32 `json:"scale_neutral_percentage_auto"`
	// Average scale neutral percentage during teleop.
	ScaleNeutralPercentageTeleop float32 `json:"scale_neutral_percentage_teleop"`
	// An array with three values, number of times a switch was owned during auto, number of opportunities to own a switch during auto, and percentage.
	SwitchOwnedCountsAuto []float32 `json:"switch_owned_counts_auto"`
	// An array with three values, number of times a unicorn match (Win + Auto Quest + Face the Boss) occurred, number of opportunities to have a unicorn match, and percentage.
	UnicornMatches []float32 `json:"unicorn_matches"`
	// Average opposing switch denail percentage for the winning alliance during teleop.
	WinningOppSwitchDenialPercentageTeleop float32 `json:"winning_opp_switch_denial_percentage_teleop"`
	// Average own switch ownership percentage for the winning alliance.
	WinningOwnSwitchOwnershipPercentage float32 `json:"winning_own_switch_ownership_percentage"`
	// Average own switch ownership percentage for the winning alliance during auto.
	WinningOwnSwitchOwnershipPercentageAuto float32 `json:"winning_own_switch_ownership_percentage_auto"`
	// Average own switch ownership percentage for the winning alliance during teleop.
	WinningOwnSwitchOwnershipPercentageTeleop float32 `json:"winning_own_switch_ownership_percentage_teleop"`
	// Average scale ownership percentage for the winning alliance.
	WinningScaleOwnershipPercentage float32 `json:"winning_scale_ownership_percentage"`
	// Average scale ownership percentage for the winning alliance during auto.
	WinningScaleOwnershipPercentageAuto float32 `json:"winning_scale_ownership_percentage_auto"`
	// Average scale ownership percentage for the winning alliance during teleop.
	WinningScaleOwnershipPercentageTeleop float32 `json:"winning_scale_ownership_percentage_teleop"`
}
