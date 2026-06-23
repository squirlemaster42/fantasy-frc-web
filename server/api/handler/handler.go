package apihandler

import (
	"server/draft"
	"server/model"
)

// Handler holds the dependencies for all API handlers.
type Handler struct {
	DraftStore    model.DraftStore
	UserStore     model.UserStore
	ApiKeyStore   model.ApiKeyStore
	TeamStore     model.TeamStore
	DraftActorMap *draft.DraftActorMap
}

func NewHandler(draftStore model.DraftStore, userStore model.UserStore, apiKeyStore model.ApiKeyStore, teamStore model.TeamStore, draftActorMap *draft.DraftActorMap) *Handler {
	return &Handler{
		DraftStore:    draftStore,
		UserStore:     userStore,
		ApiKeyStore:   apiKeyStore,
		TeamStore:     teamStore,
		DraftActorMap: draftActorMap,
	}
}
