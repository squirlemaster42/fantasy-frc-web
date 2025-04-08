# {{classname}}

All URIs are relative to *https://www.thebluealliance.com/api/v3*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetRegionalRankings**](RegionalAdvancementApi.md#GetRegionalRankings) | **Get** /regional_advancement/{year}/rankings | 

# **GetRegionalRankings**
> []RegionalRanking GetRegionalRankings(ctx, year, optional)


Gets the team rankings in the regional pool for a specific year.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **year** | **int32**| Competition Year (or Season). Must be 4 digits. | 
 **optional** | ***RegionalAdvancementApiGetRegionalRankingsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a RegionalAdvancementApiGetRegionalRankingsOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ifNoneMatch** | **optional.String**| Value of the &#x60;ETag&#x60; header in the most recently cached response by the client. | 

### Return type

[**[]RegionalRanking**](Regional_Ranking.md)

### Authorization

[apiKey](../README.md#apiKey)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

