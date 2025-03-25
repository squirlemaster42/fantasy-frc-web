# {{classname}}

All URIs are relative to *https://www.thebluealliance.com/api/v3*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetInsightsLeaderboardsYear**](InsightApi.md#GetInsightsLeaderboardsYear) | **Get** /insights/leaderboards/{year} | 
[**GetInsightsNotablesYear**](InsightApi.md#GetInsightsNotablesYear) | **Get** /insights/notables/{year} | 

# **GetInsightsLeaderboardsYear**
> []LeaderboardInsight GetInsightsLeaderboardsYear(ctx, year, optional)


Gets a list of `LeaderboardInsight` objects from a specific year. Use year=0 for overall.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **year** | **int32**| Competition Year (or Season). Must be 4 digits. | 
 **optional** | ***InsightApiGetInsightsLeaderboardsYearOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a InsightApiGetInsightsLeaderboardsYearOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]LeaderboardInsight**](LeaderboardInsight.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetInsightsNotablesYear**
> []NotablesInsight GetInsightsNotablesYear(ctx, year, optional)


Gets a list of `NotablesInsight` objects from a specific year. Use year=0 for overall.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **year** | **int32**| Competition Year (or Season). Must be 4 digits. | 
 **optional** | ***InsightApiGetInsightsNotablesYearOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a InsightApiGetInsightsNotablesYearOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]NotablesInsight**](NotablesInsight.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

