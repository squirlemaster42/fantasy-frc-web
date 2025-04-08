# EventSimple

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Key** | **string** | TBA event key with the format yyyy[EVENT_CODE], where yyyy is the year, and EVENT_CODE is the event code of the event. | [default to null]
**Name** | **string** | Official name of event on record either provided by FIRST or organizers of offseason event. | [default to null]
**EventCode** | **string** | Event short code, as provided by FIRST. | [default to null]
**EventType** | **int32** | Event Type, as defined here: https://github.com/the-blue-alliance/the-blue-alliance/blob/master/consts/event_type.py#L2 | [default to null]
**District** | [***AnyOfEventSimpleDistrict**](AnyOfEventSimpleDistrict.md) |  | [default to null]
**City** | **string** | City, town, village, etc. the event is located in. | [default to null]
**StateProv** | **string** | State or Province the event is located in. | [default to null]
**Country** | **string** | Country the event is located in. | [default to null]
**StartDate** | **string** | Event start date in &#x60;yyyy-mm-dd&#x60; format. | [default to null]
**EndDate** | **string** | Event end date in &#x60;yyyy-mm-dd&#x60; format. | [default to null]
**Year** | **int32** | Year the event data is for. | [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

