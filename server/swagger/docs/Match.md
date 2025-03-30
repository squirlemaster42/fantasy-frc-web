# Match

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Key** | **string** | TBA match key with the format &#x60;yyyy[EVENT_CODE]_[COMP_LEVEL]m[MATCH_NUMBER]&#x60;, where &#x60;yyyy&#x60; is the year, and &#x60;EVENT_CODE&#x60; is the event code of the event, &#x60;COMP_LEVEL&#x60; is (qm, ef, qf, sf, f), and &#x60;MATCH_NUMBER&#x60; is the match number in the competition level. A set number may be appended to the competition level if more than one match in required per set. | [default to null]
**CompLevel** | **string** | The competition level the match was played at. | [default to null]
**SetNumber** | **int32** | The set number in a series of matches where more than one match is required in the match series. | [default to null]
**MatchNumber** | **int32** | The match number of the match in the competition level. | [default to null]
**Alliances** | [***MatchSimpleAlliances**](Match_Simple_alliances.md) |  | [default to null]
**WinningAlliance** | **string** | The color (red/blue) of the winning alliance. Will contain an empty string in the event of no winner, or a tie. | [default to null]
**EventKey** | **string** | Event key of the event the match was played at. | [default to null]
**Time** | **int64** | UNIX timestamp (seconds since 1-Jan-1970 00:00:00) of the scheduled match time, as taken from the published schedule. | [default to null]
**ActualTime** | **int64** | UNIX timestamp (seconds since 1-Jan-1970 00:00:00) of actual match start time. | [default to null]
**PredictedTime** | **int64** | UNIX timestamp (seconds since 1-Jan-1970 00:00:00) of the TBA predicted match start time. | [default to null]
**PostResultTime** | **int64** | UNIX timestamp (seconds since 1-Jan-1970 00:00:00) when the match result was posted. | [default to null]
**ScoreBreakdown** | [***OneOfMatchScoreBreakdown**](OneOfMatchScoreBreakdown.md) | Score breakdown for auto, teleop, etc. points. Varies from year to year. May be null. | [default to null]
**Videos** | [**[]MatchVideos**](Match_videos.md) | Array of video objects associated with this match. | [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

