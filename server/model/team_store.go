package model

import "context"

type TeamStore interface {
	GetScore(ctx context.Context, tbaId string) map[string]int
	GetMatchScores(ctx context.Context, tbaId string) []MatchTeamScore
	GetTeam(ctx context.Context, tbaId string) *Team
	CreateTeam(ctx context.Context, tbaId string, name string)
}
