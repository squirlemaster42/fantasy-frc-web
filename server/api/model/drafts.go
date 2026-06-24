package apimodel

import "time"

// CreateDraftRequest is the body for POST /api/v1/drafts.
type CreateDraftRequest struct {
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	Interval    int       `json:"interval"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
}

// UpdateDraftRequest is the body for PATCH /api/v1/drafts/:id.
type UpdateDraftRequest struct {
	DisplayName string    `json:"display_name,omitempty"`
	Description string    `json:"description,omitempty"`
	Interval    int       `json:"interval,omitempty"`
	StartTime   time.Time `json:"start_time,omitempty"`
	EndTime     time.Time `json:"end_time,omitempty"`
}

// DraftResponse is the JSON representation of a draft.
type DraftResponse struct {
	Id          int        `json:"id"`
	DisplayName string     `json:"display_name"`
	Description string     `json:"description"`
	Interval    int        `json:"interval"`
	StartTime   time.Time  `json:"start_time"`
	EndTime     time.Time  `json:"end_time"`
	Owner       UserSummary `json:"owner"`
	Status      DraftState `json:"status"`
	Players     []DraftPlayerResponse `json:"players"`
	NextPick    *DraftPlayerResponse  `json:"next_pick,omitempty"`
}

// DraftPlayerResponse is the JSON representation of a draft player.
type DraftPlayerResponse struct {
	Id          int            `json:"id"`
	User        UserSummary    `json:"user"`
	PlayerOrder int16          `json:"player_order"`
	Pending     bool           `json:"pending"`
	Score       int            `json:"score"`
	Picks       []PickResponse `json:"picks,omitempty"`
}

// PickResponse is the JSON representation of a pick.
type PickResponse struct {
	Id             int        `json:"id"`
	PlayerId       int        `json:"player_id"`
	TeamKey        string     `json:"team_key,omitempty"`
	PickTime       *time.Time `json:"pick_time,omitempty"`
	AvailableTime  time.Time  `json:"available_time"`
	ExpirationTime time.Time  `json:"expiration_time"`
	Skipped        bool       `json:"skipped"`
	Score          int        `json:"score"`
}

// MakePickRequest is the body for POST /api/v1/drafts/:id/picks.
type MakePickRequest struct {
	TeamNumber int `json:"team_number"`
}

// SkipPickRequest is the body for POST /api/v1/drafts/:id/skip.
type SkipPickRequest struct {
	Skipping bool `json:"skipping"`
}

// DraftScoreResponse is returned by GET /api/v1/drafts/:id/score.
type DraftScoreResponse struct {
	DraftId int                   `json:"draft_id"`
	Status  DraftState            `json:"status"`
	Players []DraftPlayerResponse `json:"players"`
}

// MatchScore is a single match result for a team.
type MatchScore struct {
	MatchTbaId string `json:"match_tba_id"`
	Alliance   string `json:"alliance"`
	Score      int    `json:"score"`
	IsDqed     bool   `json:"is_dqed"`
}

// TeamScoreResponse is returned by GET /api/v1/team/score.
type TeamScoreResponse struct {
	TeamNumber int            `json:"team_number"`
	Scores     map[string]int `json:"scores"`
	Matches    []MatchScore   `json:"matches"`
}
