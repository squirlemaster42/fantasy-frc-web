# MatchTimeseries2018

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**EventKey** | **string** | TBA event key with the format yyyy[EVENT_CODE], where yyyy is the year, and EVENT_CODE is the event code of the event. | [optional] [default to null]
**MatchId** | **string** | Match ID consisting of the level, match number, and set number, eg &#x60;qm45&#x60; or &#x60;f1m1&#x60;. | [optional] [default to null]
**Mode** | **string** | Current mode of play, can be &#x60;pre_match&#x60;, &#x60;auto&#x60;, &#x60;telop&#x60;, or &#x60;post_match&#x60;. | [optional] [default to null]
**Play** | **int32** |  | [optional] [default to null]
**TimeRemaining** | **int32** | Amount of time remaining in the match, only valid during &#x60;auto&#x60; and &#x60;teleop&#x60; modes. | [optional] [default to null]
**BlueAutoQuest** | **int32** | 1 if the blue alliance is credited with the AUTO QUEST, 0 if not. | [optional] [default to null]
**BlueBoostCount** | **int32** | Number of POWER CUBES in the BOOST section of the blue alliance VAULT. | [optional] [default to null]
**BlueBoostPlayed** | **int32** | Returns 1 if the blue alliance BOOST was played, or 0 if not played. | [optional] [default to null]
**BlueCurrentPowerup** | **string** | Name of the current blue alliance POWER UP being played, or &#x60;null&#x60;. | [optional] [default to null]
**BlueFaceTheBoss** | **int32** | 1 if the blue alliance is credited with FACING THE BOSS, 0 if not. | [optional] [default to null]
**BlueForceCount** | **int32** | Number of POWER CUBES in the FORCE section of the blue alliance VAULT. | [optional] [default to null]
**BlueForcePlayed** | **int32** | Returns 1 if the blue alliance FORCE was played, or 0 if not played. | [optional] [default to null]
**BlueLevitateCount** | **int32** | Number of POWER CUBES in the LEVITATE section of the blue alliance VAULT. | [optional] [default to null]
**BlueLevitatePlayed** | **int32** | Returns 1 if the blue alliance LEVITATE was played, or 0 if not played. | [optional] [default to null]
**BluePowerupTimeRemaining** | **string** | Number of seconds remaining in the blue alliance POWER UP time, or 0 if none is active. | [optional] [default to null]
**BlueScaleOwned** | **int32** | 1 if the blue alliance owns the SCALE, 0 if not. | [optional] [default to null]
**BlueScore** | **int32** | Current score for the blue alliance. | [optional] [default to null]
**BlueSwitchOwned** | **int32** | 1 if the blue alliance owns their SWITCH, 0 if not. | [optional] [default to null]
**RedAutoQuest** | **int32** | 1 if the red alliance is credited with the AUTO QUEST, 0 if not. | [optional] [default to null]
**RedBoostCount** | **int32** | Number of POWER CUBES in the BOOST section of the red alliance VAULT. | [optional] [default to null]
**RedBoostPlayed** | **int32** | Returns 1 if the red alliance BOOST was played, or 0 if not played. | [optional] [default to null]
**RedCurrentPowerup** | **string** | Name of the current red alliance POWER UP being played, or &#x60;null&#x60;. | [optional] [default to null]
**RedFaceTheBoss** | **int32** | 1 if the red alliance is credited with FACING THE BOSS, 0 if not. | [optional] [default to null]
**RedForceCount** | **int32** | Number of POWER CUBES in the FORCE section of the red alliance VAULT. | [optional] [default to null]
**RedForcePlayed** | **int32** | Returns 1 if the red alliance FORCE was played, or 0 if not played. | [optional] [default to null]
**RedLevitateCount** | **int32** | Number of POWER CUBES in the LEVITATE section of the red alliance VAULT. | [optional] [default to null]
**RedLevitatePlayed** | **int32** | Returns 1 if the red alliance LEVITATE was played, or 0 if not played. | [optional] [default to null]
**RedPowerupTimeRemaining** | **string** | Number of seconds remaining in the red alliance POWER UP time, or 0 if none is active. | [optional] [default to null]
**RedScaleOwned** | **int32** | 1 if the red alliance owns the SCALE, 0 if not. | [optional] [default to null]
**RedScore** | **int32** | Current score for the red alliance. | [optional] [default to null]
**RedSwitchOwned** | **int32** | 1 if the red alliance owns their SWITCH, 0 if not. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

