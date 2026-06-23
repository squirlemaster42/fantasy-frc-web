package apimodel

import "time"

// DraftState is the JSON representation of a draft's lifecycle state.
type DraftState string

const (
	DraftStateFilling         DraftState = "Filling"
	DraftStateWaitingToStart  DraftState = "Waiting to Start"
	DraftStatePicking         DraftState = "Picking"
	DraftStateTeamsPlaying    DraftState = "Teams Playing"
	DraftStateComplete        DraftState = "Complete"
)

// UserSummary is a minimal user representation for API responses.
type UserSummary struct {
	Uuid     string `json:"uuid"`
	Username string `json:"username"`
}

// Timestamps formats RFC3339 for JSON responses.
type Timestamp = time.Time
