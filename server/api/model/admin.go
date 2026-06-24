package apimodel

import (
	"encoding/json"
	"time"
)

// AdminMakePickRequest is the body for POST /api/v1/drafts/:id/admin/make-pick.
type AdminMakePickRequest struct {
	TeamNumber int `json:"team_number"`
}

// DurationString is a time.Duration that unmarshals from a Go duration string
// such as "30m" or "1h".
type DurationString time.Duration

func (d *DurationString) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = DurationString(dur)
	return nil
}

func (d DurationString) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d DurationString) Duration() time.Duration {
	return time.Duration(d)
}

// AdminExtendTimeRequest is the body for POST /api/v1/drafts/:id/admin/extend-time.
type AdminExtendTimeRequest struct {
	Duration DurationString `json:"duration"`
}

// AdminActionResponse is returned by admin endpoints.
type AdminActionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
