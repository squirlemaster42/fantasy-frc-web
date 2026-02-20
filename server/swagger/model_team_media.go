package swagger

type TeamMedia struct {
	Details TeamMediaDetails `json:"details"`
	DirectUrl string `json:"direct_url"`
	ForeignKey string `json:"foreign_key"`
	Preferred bool `json:"preferred"`
	TeamKeys []string `json:"team_keys"`
	Type string `json:"type"`
	ViewUrl string `json:"view_url"`
}

type TeamMediaDetails struct {
	Base64Image string `json:"base64Image"`
}
