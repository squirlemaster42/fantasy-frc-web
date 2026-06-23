package apimodel

import "time"

// AdminMakePickRequest is the body for POST /api/v1/drafts/:id/admin/make-pick.
type AdminMakePickRequest struct {
	TeamNumber int `json:"team_number"`
}

// AdminExtendTimeRequest is the body for POST /api/v1/drafts/:id/admin/extend-time.
type AdminExtendTimeRequest struct {
	Duration time.Duration `json:"duration"`
}

// AdminActionResponse is returned by admin endpoints.
type AdminActionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
