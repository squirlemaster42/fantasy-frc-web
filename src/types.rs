use serde::Deserialize;
use serde::Serialize;
use serde_json::Value;

pub type Matches = Vec<Match>;

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Match {
    #[serde(rename = "actual_time")]
    pub actual_time: i64,
    pub alliances: Alliances,
    #[serde(rename = "comp_level")]
    pub comp_level: String,
    #[serde(rename = "event_key")]
    pub event_key: String,
    pub key: String,
    #[serde(rename = "match_number")]
    pub match_number: i64,
    #[serde(rename = "post_result_time")]
    pub post_result_time: i64,
    #[serde(rename = "predicted_time")]
    pub predicted_time: i64,
    #[serde(rename = "score_breakdown")]
    pub score_breakdown: ScoreBreakdown,
    #[serde(rename = "set_number")]
    pub set_number: i64,
    pub time: i64,
    pub videos: Vec<Video>,
    #[serde(rename = "winning_alliance")]
    pub winning_alliance: String,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Alliances {
    pub blue: Blue,
    pub red: Red,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Blue {
    #[serde(rename = "dq_team_keys")]
    pub dq_team_keys: Vec<Value>,
    pub score: i64,
    #[serde(rename = "surrogate_team_keys")]
    pub surrogate_team_keys: Vec<Value>,
    #[serde(rename = "team_keys")]
    pub team_keys: Vec<String>,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Red {
    #[serde(rename = "dq_team_keys")]
    pub dq_team_keys: Vec<Value>,
    pub score: i64,
    #[serde(rename = "surrogate_team_keys")]
    pub surrogate_team_keys: Vec<Value>,
    #[serde(rename = "team_keys")]
    pub team_keys: Vec<String>,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ScoreBreakdown {
    pub blue: Blue2,
    pub red: Red2,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Blue2 {
    pub activation_bonus_achieved: bool,
    pub adjust_points: i64,
    pub auto_bridge_state: String,
    pub auto_charge_station_points: i64,
    pub auto_charge_station_robot1: String,
    pub auto_charge_station_robot2: String,
    pub auto_charge_station_robot3: String,
    pub auto_community: AutoCommunity,
    pub auto_docked: bool,
    pub auto_game_piece_count: i64,
    pub auto_game_piece_points: i64,
    pub auto_mobility_points: i64,
    pub auto_points: i64,
    pub coop_game_piece_count: i64,
    pub coopertition_criteria_met: bool,
    pub end_game_bridge_state: String,
    pub end_game_charge_station_points: i64,
    pub end_game_charge_station_robot1: String,
    pub end_game_charge_station_robot2: String,
    pub end_game_charge_station_robot3: String,
    pub end_game_park_points: i64,
    pub extra_game_piece_count: i64,
    pub foul_count: i64,
    pub foul_points: i64,
    #[serde(rename = "g405Penalty")]
    pub g405penalty: bool,
    #[serde(rename = "h111Penalty")]
    pub h111penalty: bool,
    pub link_points: i64,
    pub links: Vec<Link>,
    pub mobility_robot1: String,
    pub mobility_robot2: String,
    pub mobility_robot3: String,
    pub rp: i64,
    pub sustainability_bonus_achieved: bool,
    pub tech_foul_count: i64,
    pub teleop_community: TeleopCommunity,
    pub teleop_game_piece_count: i64,
    pub teleop_game_piece_points: i64,
    pub teleop_points: i64,
    pub total_charge_station_points: i64,
    pub total_points: i64,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AutoCommunity {
    #[serde(rename = "B")]
    pub b: Vec<String>,
    #[serde(rename = "M")]
    pub m: Vec<String>,
    #[serde(rename = "T")]
    pub t: Vec<String>,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Link {
    pub nodes: Vec<i64>,
    pub row: String,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TeleopCommunity {
    #[serde(rename = "B")]
    pub b: Vec<String>,
    #[serde(rename = "M")]
    pub m: Vec<String>,
    #[serde(rename = "T")]
    pub t: Vec<String>,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Red2 {
    pub activation_bonus_achieved: bool,
    pub adjust_points: i64,
    pub auto_bridge_state: String,
    pub auto_charge_station_points: i64,
    pub auto_charge_station_robot1: String,
    pub auto_charge_station_robot2: String,
    pub auto_charge_station_robot3: String,
    pub auto_community: AutoCommunity2,
    pub auto_docked: bool,
    pub auto_game_piece_count: i64,
    pub auto_game_piece_points: i64,
    pub auto_mobility_points: i64,
    pub auto_points: i64,
    pub coop_game_piece_count: i64,
    pub coopertition_criteria_met: bool,
    pub end_game_bridge_state: String,
    pub end_game_charge_station_points: i64,
    pub end_game_charge_station_robot1: String,
    pub end_game_charge_station_robot2: String,
    pub end_game_charge_station_robot3: String,
    pub end_game_park_points: i64,
    pub extra_game_piece_count: i64,
    pub foul_count: i64,
    pub foul_points: i64,
    #[serde(rename = "g405Penalty")]
    pub g405penalty: bool,
    #[serde(rename = "h111Penalty")]
    pub h111penalty: bool,
    pub link_points: i64,
    pub links: Vec<Link2>,
    pub mobility_robot1: String,
    pub mobility_robot2: String,
    pub mobility_robot3: String,
    pub rp: i64,
    pub sustainability_bonus_achieved: bool,
    pub tech_foul_count: i64,
    pub teleop_community: TeleopCommunity2,
    pub teleop_game_piece_count: i64,
    pub teleop_game_piece_points: i64,
    pub teleop_points: i64,
    pub total_charge_station_points: i64,
    pub total_points: i64,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AutoCommunity2 {
    #[serde(rename = "B")]
    pub b: Vec<String>,
    #[serde(rename = "M")]
    pub m: Vec<String>,
    #[serde(rename = "T")]
    pub t: Vec<String>,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Link2 {
    pub nodes: Vec<i64>,
    pub row: String,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TeleopCommunity2 {
    #[serde(rename = "B")]
    pub b: Vec<String>,
    #[serde(rename = "M")]
    pub m: Vec<String>,
    #[serde(rename = "T")]
    pub t: Vec<String>,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Video {
    pub key: String,
    #[serde(rename = "type")]
    pub type_field: String,
}
