/*
 * The Blue Alliance API v3
 *
 * # Overview    Information and statistics about FIRST Robotics Competition teams and events.   # Authentication   All endpoints require an Auth Key to be passed in the header `X-TBA-Auth-Key`. If you do not have an auth key yet, you can obtain one from your [Account Page](/account).
 *
 * API version: 3.9.13
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package swagger

type TeamEventStatusRankRanking struct {
	// Number of matches played.
	MatchesPlayed int32 `json:"matches_played,omitempty"`
	// For some years, average qualification score. Can be null.
	QualAverage float64 `json:"qual_average,omitempty"`
	// Ordered list of values used to determine the rank. See the `sort_order_info` property for the name of each value.
	SortOrders []float64 `json:"sort_orders,omitempty"`
	Record *AnyOfTeamEventStatusRankRankingRecord `json:"record,omitempty"`
	// Relative rank of this team.
	Rank int32 `json:"rank,omitempty"`
	// Number of matches the team was disqualified for.
	Dq int32 `json:"dq,omitempty"`
	// TBA team key for this rank.
	TeamKey string `json:"team_key,omitempty"`
}
