package apimodel

import "time"

// CreateInviteRequest is the body for POST /api/v1/drafts/:id/invites.
type CreateInviteRequest struct {
	UserUuid string `json:"user_uuid"`
}

// InviteResponse is the JSON representation of a draft invite.
type InviteResponse struct {
	Id                 int        `json:"id"`
	DraftId            int        `json:"draft_id"`
	DraftName          string     `json:"draft_name"`
	InvitedUserUuid    string     `json:"invited_user_uuid"`
	InvitingUserUuid   string     `json:"inviting_user_uuid"`
	InvitingPlayerName string     `json:"inviting_player_name"`
	SentTime           time.Time  `json:"sent_time"`
	AcceptedTime       *time.Time `json:"accepted_time,omitempty"`
	Accepted           bool       `json:"accepted"`
}
