# {{classname}}

All URIs are relative to *https://www.thebluealliance.com/api/v3*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetDistrictAwards**](EventApi.md#GetDistrictAwards) | **Get** /district/{district_key}/awards | 
[**GetDistrictEvents**](EventApi.md#GetDistrictEvents) | **Get** /district/{district_key}/events | 
[**GetDistrictEventsKeys**](EventApi.md#GetDistrictEventsKeys) | **Get** /district/{district_key}/events/keys | 
[**GetDistrictEventsSimple**](EventApi.md#GetDistrictEventsSimple) | **Get** /district/{district_key}/events/simple | 
[**GetEvent**](EventApi.md#GetEvent) | **Get** /event/{event_key} | 
[**GetEventAdvancementPoints**](EventApi.md#GetEventAdvancementPoints) | **Get** /event/{event_key}/advancement_points | 
[**GetEventAlliances**](EventApi.md#GetEventAlliances) | **Get** /event/{event_key}/alliances | 
[**GetEventAwards**](EventApi.md#GetEventAwards) | **Get** /event/{event_key}/awards | 
[**GetEventCOPRs**](EventApi.md#GetEventCOPRs) | **Get** /event/{event_key}/coprs | 
[**GetEventDistrictPoints**](EventApi.md#GetEventDistrictPoints) | **Get** /event/{event_key}/district_points | 
[**GetEventInsights**](EventApi.md#GetEventInsights) | **Get** /event/{event_key}/insights | 
[**GetEventMatchTimeseries**](EventApi.md#GetEventMatchTimeseries) | **Get** /event/{event_key}/matches/timeseries | 
[**GetEventMatches**](EventApi.md#GetEventMatches) | **Get** /event/{event_key}/matches | 
[**GetEventMatchesKeys**](EventApi.md#GetEventMatchesKeys) | **Get** /event/{event_key}/matches/keys | 
[**GetEventMatchesSimple**](EventApi.md#GetEventMatchesSimple) | **Get** /event/{event_key}/matches/simple | 
[**GetEventOPRs**](EventApi.md#GetEventOPRs) | **Get** /event/{event_key}/oprs | 
[**GetEventPredictions**](EventApi.md#GetEventPredictions) | **Get** /event/{event_key}/predictions | 
[**GetEventRankings**](EventApi.md#GetEventRankings) | **Get** /event/{event_key}/rankings | 
[**GetEventSimple**](EventApi.md#GetEventSimple) | **Get** /event/{event_key}/simple | 
[**GetEventTeamMedia**](EventApi.md#GetEventTeamMedia) | **Get** /event/{event_key}/team_media | 
[**GetEventTeams**](EventApi.md#GetEventTeams) | **Get** /event/{event_key}/teams | 
[**GetEventTeamsKeys**](EventApi.md#GetEventTeamsKeys) | **Get** /event/{event_key}/teams/keys | 
[**GetEventTeamsSimple**](EventApi.md#GetEventTeamsSimple) | **Get** /event/{event_key}/teams/simple | 
[**GetEventTeamsStatuses**](EventApi.md#GetEventTeamsStatuses) | **Get** /event/{event_key}/teams/statuses | 
[**GetEventsByYear**](EventApi.md#GetEventsByYear) | **Get** /events/{year} | 
[**GetEventsByYearKeys**](EventApi.md#GetEventsByYearKeys) | **Get** /events/{year}/keys | 
[**GetEventsByYearSimple**](EventApi.md#GetEventsByYearSimple) | **Get** /events/{year}/simple | 
[**GetRegionalChampsPoolPoints**](EventApi.md#GetRegionalChampsPoolPoints) | **Get** /event/{event_key}/regional_champs_pool_points | 
[**GetTeamEventAwards**](EventApi.md#GetTeamEventAwards) | **Get** /team/{team_key}/event/{event_key}/awards | 
[**GetTeamEventMatches**](EventApi.md#GetTeamEventMatches) | **Get** /team/{team_key}/event/{event_key}/matches | 
[**GetTeamEventMatchesKeys**](EventApi.md#GetTeamEventMatchesKeys) | **Get** /team/{team_key}/event/{event_key}/matches/keys | 
[**GetTeamEventMatchesSimple**](EventApi.md#GetTeamEventMatchesSimple) | **Get** /team/{team_key}/event/{event_key}/matches/simple | 
[**GetTeamEventStatus**](EventApi.md#GetTeamEventStatus) | **Get** /team/{team_key}/event/{event_key}/status | 
[**GetTeamEvents**](EventApi.md#GetTeamEvents) | **Get** /team/{team_key}/events | 
[**GetTeamEventsByYear**](EventApi.md#GetTeamEventsByYear) | **Get** /team/{team_key}/events/{year} | 
[**GetTeamEventsByYearKeys**](EventApi.md#GetTeamEventsByYearKeys) | **Get** /team/{team_key}/events/{year}/keys | 
[**GetTeamEventsByYearSimple**](EventApi.md#GetTeamEventsByYearSimple) | **Get** /team/{team_key}/events/{year}/simple | 
[**GetTeamEventsKeys**](EventApi.md#GetTeamEventsKeys) | **Get** /team/{team_key}/events/keys | 
[**GetTeamEventsSimple**](EventApi.md#GetTeamEventsSimple) | **Get** /team/{team_key}/events/simple | 
[**GetTeamEventsStatusesByYear**](EventApi.md#GetTeamEventsStatusesByYear) | **Get** /team/{team_key}/events/{year}/statuses | 

# **GetDistrictAwards**
> []Award GetDistrictAwards(ctx, districtKey, optional)


Gets a list of awards in the given district.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **districtKey** | **string**| TBA District Key, eg &#x60;2016fim&#x60; | 
 **optional** | ***EventApiGetDistrictAwardsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetDistrictAwardsOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]Award**](Award.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetDistrictEvents**
> []Event GetDistrictEvents(ctx, districtKey, optional)


Gets a list of events in the given district.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **districtKey** | **string**| TBA District Key, eg &#x60;2016fim&#x60; | 
 **optional** | ***EventApiGetDistrictEventsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetDistrictEventsOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]Event**](Event.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetDistrictEventsKeys**
> []string GetDistrictEventsKeys(ctx, districtKey, optional)


Gets a list of event keys for events in the given district.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **districtKey** | **string**| TBA District Key, eg &#x60;2016fim&#x60; | 
 **optional** | ***EventApiGetDistrictEventsKeysOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetDistrictEventsKeysOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

**[]string**

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetDistrictEventsSimple**
> []EventSimple GetDistrictEventsSimple(ctx, districtKey, optional)


Gets a short-form list of events in the given district.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **districtKey** | **string**| TBA District Key, eg &#x60;2016fim&#x60; | 
 **optional** | ***EventApiGetDistrictEventsSimpleOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetDistrictEventsSimpleOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]EventSimple**](Event_Simple.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetEvent**
> Event GetEvent(ctx, eventKey, optional)


Gets an Event.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetEventOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetEventOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**Event**](Event.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetEventAdvancementPoints**
> InlineResponse2006 GetEventAdvancementPoints(ctx, eventKey, optional)


Depending on the type of event (district/regional), this will return either district points or regional CMP points

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetEventAdvancementPointsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetEventAdvancementPointsOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**InlineResponse2006**](inline_response_200_6.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetEventAlliances**
> []EliminationAlliance GetEventAlliances(ctx, eventKey, optional)


Gets a list of Elimination Alliances for the given Event.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetEventAlliancesOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetEventAlliancesOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]EliminationAlliance**](Elimination_Alliance.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetEventAwards**
> []Award GetEventAwards(ctx, eventKey, optional)


Gets a list of awards from the given event.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetEventAwardsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetEventAwardsOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]Award**](Award.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetEventCOPRs**
> InlineResponse2003 GetEventCOPRs(ctx, eventKey, optional)


Gets a set of Event Component OPRs for the given Event.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetEventCOPRsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetEventCOPRsOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**InlineResponse2003**](inline_response_200_3.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetEventDistrictPoints**
> InlineResponse2006 GetEventDistrictPoints(ctx, eventKey, optional)


Gets a list of district points for the Event. These are always calculated, regardless of event type, and may/may not be actually useful.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetEventDistrictPointsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetEventDistrictPointsOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**InlineResponse2006**](inline_response_200_6.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetEventInsights**
> InlineResponse2001 GetEventInsights(ctx, eventKey, optional)


Gets a set of Event-specific insights for the given Event.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetEventInsightsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetEventInsightsOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**InlineResponse2001**](inline_response_200_1.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetEventMatchTimeseries**
> []string GetEventMatchTimeseries(ctx, eventKey, optional)


Gets an array of Match Keys for the given event key that have timeseries data. Returns an empty array if no matches have timeseries data. *WARNING:* This is *not* official data, and is subject to a significant possibility of error, or missing data. Do not rely on this data for any purpose. In fact, pretend we made it up. *WARNING:* This endpoint and corresponding data models are under *active development* and may change at any time, including in breaking ways.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetEventMatchTimeseriesOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetEventMatchTimeseriesOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

**[]string**

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetEventMatches**
> []Match GetEventMatches(ctx, eventKey, optional)


Gets a list of matches for the given event.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetEventMatchesOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetEventMatchesOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]Match**](Match.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetEventMatchesKeys**
> []string GetEventMatchesKeys(ctx, eventKey, optional)


Gets a list of match keys for the given event.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetEventMatchesKeysOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetEventMatchesKeysOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

**[]string**

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetEventMatchesSimple**
> []MatchSimple GetEventMatchesSimple(ctx, eventKey, optional)


Gets a short-form list of matches for the given event.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetEventMatchesSimpleOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetEventMatchesSimpleOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]MatchSimple**](Match_Simple.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetEventOPRs**
> InlineResponse2002 GetEventOPRs(ctx, eventKey, optional)


Gets a set of Event OPRs (including OPR, DPR, and CCWM) for the given Event.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetEventOPRsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetEventOPRsOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**InlineResponse2002**](inline_response_200_2.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetEventPredictions**
> InlineResponse2004 GetEventPredictions(ctx, eventKey, optional)


Gets information on TBA-generated predictions for the given Event. Contains year-specific information. *WARNING* This endpoint is currently under development and may change at any time.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetEventPredictionsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetEventPredictionsOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**InlineResponse2004**](inline_response_200_4.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetEventRankings**
> InlineResponse2005 GetEventRankings(ctx, eventKey, optional)


Gets a list of team rankings for the Event.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetEventRankingsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetEventRankingsOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**InlineResponse2005**](inline_response_200_5.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetEventSimple**
> EventSimple GetEventSimple(ctx, eventKey, optional)


Gets a short-form Event.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetEventSimpleOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetEventSimpleOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**EventSimple**](Event_Simple.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetEventTeamMedia**
> []Media GetEventTeamMedia(ctx, eventKey, optional)


Gets a list of media objects that correspond to teams at this event.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetEventTeamMediaOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetEventTeamMediaOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]Media**](Media.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetEventTeams**
> []Team GetEventTeams(ctx, eventKey, optional)


Gets a list of `Team` objects that competed in the given event.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetEventTeamsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetEventTeamsOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]Team**](Team.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetEventTeamsKeys**
> []string GetEventTeamsKeys(ctx, eventKey, optional)


Gets a list of `Team` keys that competed in the given event.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetEventTeamsKeysOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetEventTeamsKeysOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

**[]string**

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetEventTeamsSimple**
> []TeamSimple GetEventTeamsSimple(ctx, eventKey, optional)


Gets a short-form list of `Team` objects that competed in the given event.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetEventTeamsSimpleOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetEventTeamsSimpleOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]TeamSimple**](Team_Simple.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetEventTeamsStatuses**
> map[string]Object GetEventTeamsStatuses(ctx, eventKey, optional)


Gets a key-value list of the event statuses for teams competing at the given event.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetEventTeamsStatusesOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetEventTeamsStatusesOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

**map[string]Object**

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetEventsByYear**
> []Event GetEventsByYear(ctx, year, optional)


Gets a list of events in the given year.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **year** | **int32**| Competition Year (or Season). Must be 4 digits. | 
 **optional** | ***EventApiGetEventsByYearOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetEventsByYearOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]Event**](Event.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetEventsByYearKeys**
> []string GetEventsByYearKeys(ctx, year, optional)


Gets a list of event keys in the given year.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **year** | **int32**| Competition Year (or Season). Must be 4 digits. | 
 **optional** | ***EventApiGetEventsByYearKeysOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetEventsByYearKeysOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

**[]string**

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetEventsByYearSimple**
> []EventSimple GetEventsByYearSimple(ctx, year, optional)


Gets a short-form list of events in the given year.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **year** | **int32**| Competition Year (or Season). Must be 4 digits. | 
 **optional** | ***EventApiGetEventsByYearSimpleOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetEventsByYearSimpleOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]EventSimple**](Event_Simple.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetRegionalChampsPoolPoints**
> InlineResponse2006 GetRegionalChampsPoolPoints(ctx, eventKey, optional)


For 2025+ Regional events, this will return points towards the Championship qualification pool.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetRegionalChampsPoolPointsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetRegionalChampsPoolPointsOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**InlineResponse2006**](inline_response_200_6.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetTeamEventAwards**
> []Award GetTeamEventAwards(ctx, teamKey, eventKey, optional)


Gets a list of awards the given team won at the given event.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **teamKey** | **string**| TBA Team Key, eg &#x60;frc254&#x60; | 
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetTeamEventAwardsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetTeamEventAwardsOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]Award**](Award.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetTeamEventMatches**
> []Match GetTeamEventMatches(ctx, teamKey, eventKey, optional)


Gets a list of matches for the given team and event.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **teamKey** | **string**| TBA Team Key, eg &#x60;frc254&#x60; | 
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetTeamEventMatchesOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetTeamEventMatchesOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]Match**](Match.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetTeamEventMatchesKeys**
> []string GetTeamEventMatchesKeys(ctx, teamKey, eventKey, optional)


Gets a list of match keys for matches for the given team and event.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **teamKey** | **string**| TBA Team Key, eg &#x60;frc254&#x60; | 
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetTeamEventMatchesKeysOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetTeamEventMatchesKeysOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

**[]string**

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetTeamEventMatchesSimple**
> []Match GetTeamEventMatchesSimple(ctx, teamKey, eventKey, optional)


Gets a short-form list of matches for the given team and event.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **teamKey** | **string**| TBA Team Key, eg &#x60;frc254&#x60; | 
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetTeamEventMatchesSimpleOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetTeamEventMatchesSimpleOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]Match**](Match.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetTeamEventStatus**
> InlineResponse200 GetTeamEventStatus(ctx, teamKey, eventKey, optional)


Gets the competition rank and status of the team at the given event.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **teamKey** | **string**| TBA Team Key, eg &#x60;frc254&#x60; | 
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***EventApiGetTeamEventStatusOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetTeamEventStatusOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**InlineResponse200**](inline_response_200.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetTeamEvents**
> []Event GetTeamEvents(ctx, teamKey, optional)


Gets a list of all events this team has competed at.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **teamKey** | **string**| TBA Team Key, eg &#x60;frc254&#x60; | 
 **optional** | ***EventApiGetTeamEventsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetTeamEventsOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]Event**](Event.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetTeamEventsByYear**
> []Event GetTeamEventsByYear(ctx, teamKey, year, optional)


Gets a list of events this team has competed at in the given year.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **teamKey** | **string**| TBA Team Key, eg &#x60;frc254&#x60; | 
  **year** | **int32**| Competition Year (or Season). Must be 4 digits. | 
 **optional** | ***EventApiGetTeamEventsByYearOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetTeamEventsByYearOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]Event**](Event.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetTeamEventsByYearKeys**
> []string GetTeamEventsByYearKeys(ctx, teamKey, year, optional)


Gets a list of the event keys for events this team has competed at in the given year.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **teamKey** | **string**| TBA Team Key, eg &#x60;frc254&#x60; | 
  **year** | **int32**| Competition Year (or Season). Must be 4 digits. | 
 **optional** | ***EventApiGetTeamEventsByYearKeysOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetTeamEventsByYearKeysOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

**[]string**

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetTeamEventsByYearSimple**
> []EventSimple GetTeamEventsByYearSimple(ctx, teamKey, year, optional)


Gets a short-form list of events this team has competed at in the given year.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **teamKey** | **string**| TBA Team Key, eg &#x60;frc254&#x60; | 
  **year** | **int32**| Competition Year (or Season). Must be 4 digits. | 
 **optional** | ***EventApiGetTeamEventsByYearSimpleOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetTeamEventsByYearSimpleOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]EventSimple**](Event_Simple.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetTeamEventsKeys**
> []string GetTeamEventsKeys(ctx, teamKey, optional)


Gets a list of the event keys for all events this team has competed at.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **teamKey** | **string**| TBA Team Key, eg &#x60;frc254&#x60; | 
 **optional** | ***EventApiGetTeamEventsKeysOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetTeamEventsKeysOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

**[]string**

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetTeamEventsSimple**
> []EventSimple GetTeamEventsSimple(ctx, teamKey, optional)


Gets a short-form list of all events this team has competed at.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **teamKey** | **string**| TBA Team Key, eg &#x60;frc254&#x60; | 
 **optional** | ***EventApiGetTeamEventsSimpleOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetTeamEventsSimpleOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]EventSimple**](Event_Simple.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetTeamEventsStatusesByYear**
> map[string]Object GetTeamEventsStatusesByYear(ctx, teamKey, year, optional)


Gets a key-value list of the event statuses for events this team has competed at in the given year.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **teamKey** | **string**| TBA Team Key, eg &#x60;frc254&#x60; | 
  **year** | **int32**| Competition Year (or Season). Must be 4 digits. | 
 **optional** | ***EventApiGetTeamEventsStatusesByYearOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a EventApiGetTeamEventsStatusesByYearOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

**map[string]Object**

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

