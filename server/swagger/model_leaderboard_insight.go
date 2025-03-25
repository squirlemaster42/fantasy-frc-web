/*
 * The Blue Alliance API v3
 *
 * # Overview    Information and statistics about FIRST Robotics Competition teams and events.   # Authentication   All endpoints require an Auth Key to be passed in the header `X-TBA-Auth-Key`. If you do not have an auth key yet, you can obtain one from your [Account Page](/account).
 *
 * API version: 3.9.13
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package swagger

type LeaderboardInsight struct {
	Data *LeaderboardInsightData `json:"data"`
	// Name of the insight.
	Name string `json:"name"`
	// Year the insight was measured in (year=0 for overall insights).
	Year int32 `json:"year"`
}
