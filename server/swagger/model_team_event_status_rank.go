/*
 * The Blue Alliance API v3
 *
 * # Overview    Information and statistics about FIRST Robotics Competition teams and events.   # Authentication   All endpoints require an Auth Key to be passed in the header `X-TBA-Auth-Key`. If you do not have an auth key yet, you can obtain one from your [Account Page](/account).
 *
 * API version: 3.9.13
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package swagger

type TeamEventStatusRank struct {
	// Number of teams ranked.
	NumTeams int32 `json:"num_teams,omitempty"`
	Ranking *TeamEventStatusRankRanking `json:"ranking,omitempty"`
	// Ordered list of names corresponding to the elements of the `sort_orders` array.
	SortOrderInfo []TeamEventStatusRankSortOrderInfo `json:"sort_order_info,omitempty"`
	Status string `json:"status,omitempty"`
}
