# Event

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Key** | **string** | TBA event key with the format yyyy[EVENT_CODE], where yyyy is the year, and EVENT_CODE is the event code of the event. | [default to null]
**Name** | **string** | Official name of event on record either provided by FIRST or organizers of offseason event. | [default to null]
**EventCode** | **string** | Event short code, as provided by FIRST. | [default to null]
**EventType** | **int32** | Event Type, as defined here: https://github.com/the-blue-alliance/the-blue-alliance/blob/master/consts/event_type.py#L2 | [default to null]
**District** | [***AnyOfEventDistrict**](AnyOfEventDistrict.md) |  | [default to null]
**City** | **string** | City, town, village, etc. the event is located in. | [default to null]
**StateProv** | **string** | State or Province the event is located in. | [default to null]
**Country** | **string** | Country the event is located in. | [default to null]
**StartDate** | **string** | Event start date in &#x60;yyyy-mm-dd&#x60; format. | [default to null]
**EndDate** | **string** | Event end date in &#x60;yyyy-mm-dd&#x60; format. | [default to null]
**Year** | **int32** | Year the event data is for. | [default to null]
**ShortName** | **string** | Same as &#x60;name&#x60; but doesn&#x27;t include event specifiers, such as &#x27;Regional&#x27; or &#x27;District&#x27;. May be null. | [default to null]
**EventTypeString** | **string** | Event Type, eg Regional, District, or Offseason. | [default to null]
**Week** | **int32** | Week of the event relative to the first official season event, zero-indexed. Only valid for Regionals, Districts, and District Championships. Null otherwise. (Eg. A season with a week 0 &#x27;preseason&#x27; event does not count, and week 1 events will show 0 here. Seasons with a week 0.5 regional event will show week 0 for those event(s) and week 1 for week 1 events and so on.) | [default to null]
**Address** | **string** | Address of the event&#x27;s venue, if available. | [default to null]
**PostalCode** | **string** | Postal code from the event address. | [default to null]
**GmapsPlaceId** | **string** | Google Maps Place ID for the event address. | [default to null]
**GmapsUrl** | **string** | Link to address location on Google Maps. | [default to null]
**Lat** | **float64** | Latitude for the event address. | [default to null]
**Lng** | **float64** | Longitude for the event address. | [default to null]
**LocationName** | **string** | Name of the location at the address for the event, eg. Blue Alliance High School. | [default to null]
**Timezone** | **string** | Timezone name. | [default to null]
**Website** | **string** | The event&#x27;s website, if any. | [default to null]
**FirstEventId** | **string** | The FIRST internal Event ID, used to link to the event on the FRC webpage. | [default to null]
**FirstEventCode** | **string** | Public facing event code used by FIRST (on frc-events.firstinspires.org, for example) | [default to null]
**Webcasts** | [**[]Webcast**](Webcast.md) |  | [default to null]
**DivisionKeys** | **[]string** | An array of event keys for the divisions at this event. | [default to null]
**ParentEventKey** | **string** | The TBA Event key that represents the event&#x27;s parent. Used to link back to the event from a division event. It is also the inverse relation of &#x60;divison_keys&#x60;. | [default to null]
**PlayoffType** | **int32** | Playoff Type, as defined here: https://github.com/the-blue-alliance/the-blue-alliance/blob/master/consts/playoff_type.py#L4, or null. | [default to null]
**PlayoffTypeString** | **string** | String representation of the &#x60;playoff_type&#x60;, or null. | [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

