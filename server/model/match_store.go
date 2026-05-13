package model

import (
	"context"
)

type MatchStore interface {
	AddMatch(ctx context.Context, tbaId string)
	UpdateScore(ctx context.Context, tbaId string, redScore int, blueScore int)
	GetMatch(ctx context.Context, tbaId string) *Match
}
