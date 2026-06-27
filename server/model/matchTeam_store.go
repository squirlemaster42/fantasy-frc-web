package model

import "context"

type MatchTeamStore interface {
	AssociateTeam(ctx context.Context, matchTbaId string, teamTbaId string, alliance string, isDqed bool) error
}
