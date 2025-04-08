# {{classname}}

All URIs are relative to *https://www.thebluealliance.com/api/v3*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetEventMatchTimeseries**](MatchApi.md#GetEventMatchTimeseries) | **Get** /event/{event_key}/matches/timeseries | 
[**GetEventMatches**](MatchApi.md#GetEventMatches) | **Get** /event/{event_key}/matches | 
[**GetEventMatchesKeys**](MatchApi.md#GetEventMatchesKeys) | **Get** /event/{event_key}/matches/keys | 
[**GetEventMatchesSimple**](MatchApi.md#GetEventMatchesSimple) | **Get** /event/{event_key}/matches/simple | 
[**GetMatch**](MatchApi.md#GetMatch) | **Get** /match/{match_key} | 
[**GetMatchSimple**](MatchApi.md#GetMatchSimple) | **Get** /match/{match_key}/simple | 
[**GetMatchTimeseries**](MatchApi.md#GetMatchTimeseries) | **Get** /match/{match_key}/timeseries | 
[**GetMatchZebra**](MatchApi.md#GetMatchZebra) | **Get** /match/{match_key}/zebra_motionworks | 
[**GetTeamEventMatches**](MatchApi.md#GetTeamEventMatches) | **Get** /team/{team_key}/event/{event_key}/matches | 
[**GetTeamEventMatchesKeys**](MatchApi.md#GetTeamEventMatchesKeys) | **Get** /team/{team_key}/event/{event_key}/matches/keys | 
[**GetTeamEventMatchesSimple**](MatchApi.md#GetTeamEventMatchesSimple) | **Get** /team/{team_key}/event/{event_key}/matches/simple | 
[**GetTeamMatchesByYear**](MatchApi.md#GetTeamMatchesByYear) | **Get** /team/{team_key}/matches/{year} | 
[**GetTeamMatchesByYearKeys**](MatchApi.md#GetTeamMatchesByYearKeys) | **Get** /team/{team_key}/matches/{year}/keys | 
[**GetTeamMatchesByYearSimple**](MatchApi.md#GetTeamMatchesByYearSimple) | **Get** /team/{team_key}/matches/{year}/simple | 

# **GetEventMatchTimeseries**
> []string GetEventMatchTimeseries(ctx, eventKey, optional)


Gets an array of Match Keys for the given event key that have timeseries data. Returns an empty array if no matches have timeseries data. *WARNING:* This is *not* official data, and is subject to a significant possibility of error, or missing data. Do not rely on this data for any purpose. In fact, pretend we made it up. *WARNING:* This endpoint and corresponding data models are under *active development* and may change at any time, including in breaking ways.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **eventKey** | **string**| TBA Event Key, eg &#x60;2016nytr&#x60; | 
 **optional** | ***MatchApiGetEventMatchTimeseriesOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a MatchApiGetEventMatchTimeseriesOpts struct
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
 **optional** | ***MatchApiGetEventMatchesOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a MatchApiGetEventMatchesOpts struct
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
 **optional** | ***MatchApiGetEventMatchesKeysOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a MatchApiGetEventMatchesKeysOpts struct
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
 **optional** | ***MatchApiGetEventMatchesSimpleOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a MatchApiGetEventMatchesSimpleOpts struct
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

# **GetMatch**
> Match GetMatch(ctx, matchKey, optional)


Gets a `Match` object for the given match key.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **matchKey** | **string**| TBA Match Key, eg &#x60;2016nytr_qm1&#x60; | 
 **optional** | ***MatchApiGetMatchOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a MatchApiGetMatchOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**Match**](Match.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetMatchSimple**
> MatchSimple GetMatchSimple(ctx, matchKey, optional)


Gets a short-form `Match` object for the given match key.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **matchKey** | **string**| TBA Match Key, eg &#x60;2016nytr_qm1&#x60; | 
 **optional** | ***MatchApiGetMatchSimpleOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a MatchApiGetMatchSimpleOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**MatchSimple**](Match_Simple.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetMatchTimeseries**
> []interface{} GetMatchTimeseries(ctx, matchKey, optional)


Gets an array of game-specific Match Timeseries objects for the given match key or an empty array if not available. *WARNING:* This is *not* official data, and is subject to a significant possibility of error, or missing data. Do not rely on this data for any purpose. In fact, pretend we made it up. *WARNING:* This endpoint and corresponding data models are under *active development* and may change at any time, including in breaking ways.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **matchKey** | **string**| TBA Match Key, eg &#x60;2016nytr_qm1&#x60; | 
 **optional** | ***MatchApiGetMatchTimeseriesOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a MatchApiGetMatchTimeseriesOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]interface{}**](interface{}.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetMatchZebra**
> Zebra GetMatchZebra(ctx, matchKey, optional)


Gets Zebra MotionWorks data for a Match for the given match key.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **matchKey** | **string**| TBA Match Key, eg &#x60;2016nytr_qm1&#x60; | 
 **optional** | ***MatchApiGetMatchZebraOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a MatchApiGetMatchZebraOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**Zebra**](Zebra.md)

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
 **optional** | ***MatchApiGetTeamEventMatchesOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a MatchApiGetTeamEventMatchesOpts struct
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
 **optional** | ***MatchApiGetTeamEventMatchesKeysOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a MatchApiGetTeamEventMatchesKeysOpts struct
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
 **optional** | ***MatchApiGetTeamEventMatchesSimpleOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a MatchApiGetTeamEventMatchesSimpleOpts struct
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

# **GetTeamMatchesByYear**
> []Match GetTeamMatchesByYear(ctx, teamKey, year, optional)


Gets a list of matches for the given team and year.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **teamKey** | **string**| TBA Team Key, eg &#x60;frc254&#x60; | 
  **year** | **int32**| Competition Year (or Season). Must be 4 digits. | 
 **optional** | ***MatchApiGetTeamMatchesByYearOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a MatchApiGetTeamMatchesByYearOpts struct
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

# **GetTeamMatchesByYearKeys**
> []string GetTeamMatchesByYearKeys(ctx, teamKey, year, optional)


Gets a list of match keys for matches for the given team and year.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **teamKey** | **string**| TBA Team Key, eg &#x60;frc254&#x60; | 
  **year** | **int32**| Competition Year (or Season). Must be 4 digits. | 
 **optional** | ***MatchApiGetTeamMatchesByYearKeysOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a MatchApiGetTeamMatchesByYearKeysOpts struct
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

# **GetTeamMatchesByYearSimple**
> []MatchSimple GetTeamMatchesByYearSimple(ctx, teamKey, year, optional)


Gets a short-form list of matches for the given team and year.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **teamKey** | **string**| TBA Team Key, eg &#x60;frc254&#x60; | 
  **year** | **int32**| Competition Year (or Season). Must be 4 digits. | 
 **optional** | ***MatchApiGetTeamMatchesByYearSimpleOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a MatchApiGetTeamMatchesByYearSimpleOpts struct
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

