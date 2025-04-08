/*
 * The Blue Alliance API v3
 *
 * # Overview    Information and statistics about FIRST Robotics Competition teams and events.   # Authentication   All endpoints require an Auth Key to be passed in the header `X-TBA-Auth-Key`. If you do not have an auth key yet, you can obtain one from your [Account Page](/account).
 *
 * API version: 3.9.13
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package swagger

type MatchAlliance struct {
	// Score for this alliance. Will be null or -1 for an unplayed match.
	Score int32 `json:"score"`
	TeamKeys []string `json:"team_keys"`
	// TBA team keys (eg `frc254`) of any teams playing as a surrogate.
	SurrogateTeamKeys []string `json:"surrogate_team_keys"`
	// TBA team keys (eg `frc254`) of any disqualified teams.
	DqTeamKeys []string `json:"dq_team_keys"`
}
