/*
 * The Blue Alliance API v3
 *
 * # Overview    Information and statistics about FIRST Robotics Competition teams and events.   # Authentication   All endpoints require an Auth Key to be passed in the header `X-TBA-Auth-Key`. If you do not have an auth key yet, you can obtain one from your [Account Page](/account).
 *
 * API version: 3.9.13
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package swagger

type DistrictRankingEventPoints struct {
	// `true` if this event is a District Championship event.
	DistrictCmp bool `json:"district_cmp"`
	// Total points awarded at this event.
	Total int32 `json:"total"`
	// Points awarded for alliance selection.
	AlliancePoints int32 `json:"alliance_points"`
	// Points awarded for elimination match performance.
	ElimPoints int32 `json:"elim_points"`
	// Points awarded for event awards.
	AwardPoints int32 `json:"award_points"`
	// TBA Event key for this event.
	EventKey string `json:"event_key"`
	// Points awarded for qualification match performance.
	QualPoints int32 `json:"qual_points"`
}
