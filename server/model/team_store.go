package model

import "context"

type TeamStore interface {
	GetScore(ctx context.Context, tbaId string) (map[string]int, error)
	GetMatchScores(ctx context.Context, tbaId string) ([]MatchTeamScore, error)
	GetTeam(ctx context.Context, tbaId string) (*Team, error)
	CreateTeam(ctx context.Context, tbaId string, name string) error
}
