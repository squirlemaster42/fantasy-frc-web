package model

import "context"

type MatchTeamStore interface {
	AssocateTeam(ctx context.Context, matchTbaId string, teamTbaId string, alliance string, isDqed bool) error
}
