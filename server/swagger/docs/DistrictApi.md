# {{classname}}

All URIs are relative to *https://www.thebluealliance.com/api/v3*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetDistrictAwards**](DistrictApi.md#GetDistrictAwards) | **Get** /district/{district_key}/awards | 
[**GetDistrictEvents**](DistrictApi.md#GetDistrictEvents) | **Get** /district/{district_key}/events | 
[**GetDistrictEventsKeys**](DistrictApi.md#GetDistrictEventsKeys) | **Get** /district/{district_key}/events/keys | 
[**GetDistrictEventsSimple**](DistrictApi.md#GetDistrictEventsSimple) | **Get** /district/{district_key}/events/simple | 
[**GetDistrictHistory**](DistrictApi.md#GetDistrictHistory) | **Get** /district/{district_abbreviation}/history | 
[**GetDistrictRankings**](DistrictApi.md#GetDistrictRankings) | **Get** /district/{district_key}/rankings | 
[**GetDistrictTeams**](DistrictApi.md#GetDistrictTeams) | **Get** /district/{district_key}/teams | 
[**GetDistrictTeamsKeys**](DistrictApi.md#GetDistrictTeamsKeys) | **Get** /district/{district_key}/teams/keys | 
[**GetDistrictTeamsSimple**](DistrictApi.md#GetDistrictTeamsSimple) | **Get** /district/{district_key}/teams/simple | 
[**GetDistrictsByYear**](DistrictApi.md#GetDistrictsByYear) | **Get** /districts/{year} | 
[**GetEventDistrictPoints**](DistrictApi.md#GetEventDistrictPoints) | **Get** /event/{event_key}/district_points | 
[**GetTeamDistricts**](DistrictApi.md#GetTeamDistricts) | **Get** /team/{team_key}/districts | 

# **GetDistrictAwards**
> []Award GetDistrictAwards(ctx, districtKey, optional)


Gets a list of awards in the given district.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **districtKey** | **string**| TBA District Key, eg &#x60;2016fim&#x60; | 
 **optional** | ***DistrictApiGetDistrictAwardsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a DistrictApiGetDistrictAwardsOpts struct
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
 **optional** | ***DistrictApiGetDistrictEventsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a DistrictApiGetDistrictEventsOpts struct
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
 **optional** | ***DistrictApiGetDistrictEventsKeysOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a DistrictApiGetDistrictEventsKeysOpts struct
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
 **optional** | ***DistrictApiGetDistrictEventsSimpleOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a DistrictApiGetDistrictEventsSimpleOpts struct
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

# **GetDistrictHistory**
> []DistrictList GetDistrictHistory(ctx, districtAbbreviation, optional)


Gets a list of District objects with the given district abbreviation. This accounts for district abbreviation changes, such as MAR to FMA.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **districtAbbreviation** | **string**| District abbreviation, eg &#x60;ne&#x60; or &#x60;fim&#x60; | 
 **optional** | ***DistrictApiGetDistrictHistoryOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a DistrictApiGetDistrictHistoryOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]DistrictList**](District_List.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetDistrictRankings**
> []DistrictRanking GetDistrictRankings(ctx, districtKey, optional)


Gets a list of team district rankings for the given district.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **districtKey** | **string**| TBA District Key, eg &#x60;2016fim&#x60; | 
 **optional** | ***DistrictApiGetDistrictRankingsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a DistrictApiGetDistrictRankingsOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]DistrictRanking**](District_Ranking.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetDistrictTeams**
> []Team GetDistrictTeams(ctx, districtKey, optional)


Gets a list of `Team` objects that competed in events in the given district.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **districtKey** | **string**| TBA District Key, eg &#x60;2016fim&#x60; | 
 **optional** | ***DistrictApiGetDistrictTeamsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a DistrictApiGetDistrictTeamsOpts struct
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

# **GetDistrictTeamsKeys**
> []string GetDistrictTeamsKeys(ctx, districtKey, optional)


Gets a list of `Team` objects that competed in events in the given district.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **districtKey** | **string**| TBA District Key, eg &#x60;2016fim&#x60; | 
 **optional** | ***DistrictApiGetDistrictTeamsKeysOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a DistrictApiGetDistrictTeamsKeysOpts struct
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

# **GetDistrictTeamsSimple**
> []TeamSimple GetDistrictTeamsSimple(ctx, districtKey, optional)


Gets a short-form list of `Team` objects that competed in events in the given district.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **districtKey** | **string**| TBA District Key, eg &#x60;2016fim&#x60; | 
 **optional** | ***DistrictApiGetDistrictTeamsSimpleOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a DistrictApiGetDistrictTeamsSimpleOpts struct
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

# **GetDistrictsByYear**
> []DistrictList GetDistrictsByYear(ctx, year, optional)


Gets a list of districts and their corresponding district key, for the given year.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **year** | **int32**| Competition Year (or Season). Must be 4 digits. | 
 **optional** | ***DistrictApiGetDistrictsByYearOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a DistrictApiGetDistrictsByYearOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]DistrictList**](District_List.md)

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
 **optional** | ***DistrictApiGetEventDistrictPointsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a DistrictApiGetEventDistrictPointsOpts struct
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

# **GetTeamDistricts**
> []DistrictList GetTeamDistricts(ctx, teamKey, optional)


Gets an array of districts representing each year the team was in a district. Will return an empty array if the team was never in a district.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **teamKey** | **string**| TBA Team Key, eg &#x60;frc254&#x60; | 
 **optional** | ***DistrictApiGetTeamDistrictsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a DistrictApiGetTeamDistrictsOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]DistrictList**](District_List.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

