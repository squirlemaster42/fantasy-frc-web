package draft

import (
	"testing"
	"time"

	"server/model"
	"server/model/mocks"
	"server/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDraftActorMap_GetActor_CacheHit(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("GetDraft", mock.Anything, 1).Return(model.DraftModel{Id: 1}, nil).Once()

	actorMap := NewDraftActorMap(mockStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 2)

	ctx := t.Context()
	actor1, err := actorMap.GetActor(ctx, 1)
	require.NoError(t, err)

	actor2, err := actorMap.GetActor(ctx, 1)
	require.NoError(t, err)

	assert.Same(t, actor1, actor2)
	mockStore.AssertExpectations(t)
}

func TestDraftActorMap_GetActor_EvictsOldest(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("GetDraft", mock.Anything, 1).Return(model.DraftModel{Id: 1}, nil).Once()
	mockStore.On("GetDraft", mock.Anything, 2).Return(model.DraftModel{Id: 2}, nil).Once()

	actorMap := NewDraftActorMap(mockStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 1)

	ctx := t.Context()
	actor1, err := actorMap.GetActor(ctx, 1)
	require.NoError(t, err)
	require.False(t, actor1.IsShutdown())

	_, err = actorMap.GetActor(ctx, 2)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return actor1.IsShutdown()
	}, time.Second, 10*time.Millisecond, "evicted actor should shut down")

	mockStore.AssertExpectations(t)
}

func TestDraftActorMap_ShutdownActor_RemovesFromCache(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("GetDraft", mock.Anything, 1).Return(model.DraftModel{Id: 1}, nil).Twice()

	actorMap := NewDraftActorMap(mockStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 2)

	ctx := t.Context()
	actor1, err := actorMap.GetActor(ctx, 1)
	require.NoError(t, err)

	err = ShutdownActor(actorMap, ctx, 1)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return actor1.IsShutdown()
	}, time.Second, 10*time.Millisecond, "shutdown actor should shut down")

	actor2, err := actorMap.GetActor(ctx, 1)
	require.NoError(t, err)
	assert.NotSame(t, actor1, actor2)

	mockStore.AssertExpectations(t)
}
