package model

import (
	"context"
)

type MatchStore interface {
	AddMatch(ctx context.Context, tbaId string) error
	UpdateScore(ctx context.Context, tbaId string, redScore int, blueScore int) error
	GetMatch(ctx context.Context, tbaId string) (*Match, error)
}
