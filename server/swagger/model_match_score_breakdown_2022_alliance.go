/*
 * The Blue Alliance API v3
 *
 * # Overview    Information and statistics about FIRST Robotics Competition teams and events.   # Authentication   All endpoints require an Auth Key to be passed in the header `X-TBA-Auth-Key`. If you do not have an auth key yet, you can obtain one from your [Account Page](/account).
 *
 * API version: 3.9.13
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package swagger

type MatchScoreBreakdown2022Alliance struct {
	TaxiRobot1 string `json:"taxiRobot1,omitempty"`
	EndgameRobot1 string `json:"endgameRobot1,omitempty"`
	TaxiRobot2 string `json:"taxiRobot2,omitempty"`
	EndgameRobot2 string `json:"endgameRobot2,omitempty"`
	TaxiRobot3 string `json:"taxiRobot3,omitempty"`
	EndgameRobot3 string `json:"endgameRobot3,omitempty"`
	AutoCargoLowerNear int32 `json:"autoCargoLowerNear,omitempty"`
	AutoCargoLowerFar int32 `json:"autoCargoLowerFar,omitempty"`
	AutoCargoLowerBlue int32 `json:"autoCargoLowerBlue,omitempty"`
	AutoCargoLowerRed int32 `json:"autoCargoLowerRed,omitempty"`
	AutoCargoUpperNear int32 `json:"autoCargoUpperNear,omitempty"`
	AutoCargoUpperFar int32 `json:"autoCargoUpperFar,omitempty"`
	AutoCargoUpperBlue int32 `json:"autoCargoUpperBlue,omitempty"`
	AutoCargoUpperRed int32 `json:"autoCargoUpperRed,omitempty"`
	AutoCargoTotal int32 `json:"autoCargoTotal,omitempty"`
	TeleopCargoLowerNear int32 `json:"teleopCargoLowerNear,omitempty"`
	TeleopCargoLowerFar int32 `json:"teleopCargoLowerFar,omitempty"`
	TeleopCargoLowerBlue int32 `json:"teleopCargoLowerBlue,omitempty"`
	TeleopCargoLowerRed int32 `json:"teleopCargoLowerRed,omitempty"`
	TeleopCargoUpperNear int32 `json:"teleopCargoUpperNear,omitempty"`
	TeleopCargoUpperFar int32 `json:"teleopCargoUpperFar,omitempty"`
	TeleopCargoUpperBlue int32 `json:"teleopCargoUpperBlue,omitempty"`
	TeleopCargoUpperRed int32 `json:"teleopCargoUpperRed,omitempty"`
	TeleopCargoTotal int32 `json:"teleopCargoTotal,omitempty"`
	MatchCargoTotal int32 `json:"matchCargoTotal,omitempty"`
	AutoTaxiPoints int32 `json:"autoTaxiPoints,omitempty"`
	AutoCargoPoints int32 `json:"autoCargoPoints,omitempty"`
	AutoPoints int32 `json:"autoPoints,omitempty"`
	QuintetAchieved bool `json:"quintetAchieved,omitempty"`
	TeleopCargoPoints int32 `json:"teleopCargoPoints,omitempty"`
	EndgamePoints int32 `json:"endgamePoints,omitempty"`
	TeleopPoints int32 `json:"teleopPoints,omitempty"`
	CargoBonusRankingPoint bool `json:"cargoBonusRankingPoint,omitempty"`
	HangarBonusRankingPoint bool `json:"hangarBonusRankingPoint,omitempty"`
	FoulCount int32 `json:"foulCount,omitempty"`
	TechFoulCount int32 `json:"techFoulCount,omitempty"`
	AdjustPoints int32 `json:"adjustPoints,omitempty"`
	FoulPoints int32 `json:"foulPoints,omitempty"`
	Rp int32 `json:"rp,omitempty"`
	TotalPoints int32 `json:"totalPoints,omitempty"`
}
