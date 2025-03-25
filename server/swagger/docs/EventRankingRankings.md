# EventRankingRankings

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**MatchesPlayed** | **int32** | Number of matches played by this team. | [default to null]
**QualAverage** | **int32** | The average match score during qualifications. Year specific. May be null if not relevant for a given year. | [default to null]
**ExtraStats** | **[]float64** | Additional special data on the team&#x27;s performance calculated by TBA. | [default to null]
**SortOrders** | **[]float64** | Additional year-specific information, may be null. See parent &#x60;sort_order_info&#x60; for details. | [default to null]
**Record** | [***AnyOfEventRankingRankingsRecord**](AnyOfEventRankingRankingsRecord.md) |  | [default to null]
**Rank** | **int32** | The team&#x27;s rank at the event as provided by FIRST. | [default to null]
**Dq** | **int32** | Number of times disqualified. | [default to null]
**TeamKey** | **string** | The team with this rank. | [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

