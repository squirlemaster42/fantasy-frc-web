package apimodel

// UserSearchResponse is returned by GET /api/v1/users/search.
type UserSearchResponse struct {
	Users []UserSummary `json:"users"`
}
