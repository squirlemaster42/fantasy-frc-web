use reqwest;
use crate::types::Matches;

pub async fn make_tba_request(url: &str) -> Result<String, reqwest::Error> {
    let client = reqwest::Client::new();
    let res = client
        .get(url)
        .header(
            "X-TBA-Auth-Key",
            std::env::var("TBA_TOKEN").expect("TBA TOKEN not found"),
        )
        .send()
        .await?
        .text()
        .await?;

    return Ok(res);
}

pub async fn make_tba_match_list_request(event_str: &str) -> Matches {
    let mut req_string: String = "https://www.thebluealliance.com/api/v3/event/".to_owned();
    req_string.push_str(event_str);
    req_string.push_str("/matches");

    let tba_json = make_tba_request(req_string.as_str()).await.unwrap();

    let parsed_match: Matches = serde_json::from_str(tba_json.as_str()).expect("Json not formed correctly");

    return parsed_match;
}

pub async fn make_tba_match_list_for_team_request(event_str: &str, team_str: &str) -> Matches {
    //TODO Complete
}

pub async fn make_tba_events_for_team_request(team_str: &str) -> Events {
    //TODO Complete
}

pub async fn make_match_request(match_str: &str) -> Match {
    //TODO Complete
}

pub async fn make_tba_match_keys_for_team(event_str: &str, team_str: &str) -> Vec<String> {
    //TODO Complete
}

pub async fn make_tba_match_keys_for_team_for_year_request(team_str: &str, year: &str) -> Vec<String> {
    //TODO Complete
}

pub async fn make_team_event_status_request(event_str: &str, team_str: &str) -> Vec<String> {
    //TODO Complete
}
