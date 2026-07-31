package scorer

import (
	"context"
	"server/swagger"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchQueueOrdering(t *testing.T) {
    queue := NewMatchQueue()
    queue.PushMatch(swagger.Match{
        Key: "2024cur_qm1",
    })
    queue.PushMatch(swagger.Match{
        Key: "2024cur_qm72",
    })
    queue.PushMatch(swagger.Match{
        Key: "2024cur_qm112",
    })
    queue.PushMatch(swagger.Match{
        Key: "2024cur_sf2m1",
    })
    queue.PushMatch(swagger.Match{
        Key: "2024cur_sf9m1",
    })
    queue.PushMatch(swagger.Match{
        Key: "2024cur_sf12m1",
    })
    queue.PushMatch(swagger.Match{
        Key: "2024cur_f1m1",
    })
    queue.PushMatch(swagger.Match{
        Key: "2024cur_f1m2",
    })
    ctx := context.Background()
    match, err := queue.PopMatch(ctx)
    assert.NoError(t, err)
    assert.Equal(t, "2024cur_qm1", match.Key)
    match, err = queue.PopMatch(ctx)
    assert.NoError(t, err)
    assert.Equal(t, "2024cur_qm72", match.Key)
    match, err = queue.PopMatch(ctx)
    assert.NoError(t, err)
    assert.Equal(t, "2024cur_qm112", match.Key)
    match, err = queue.PopMatch(ctx)
    assert.NoError(t, err)
    assert.Equal(t, "2024cur_sf2m1", match.Key)
    match, err = queue.PopMatch(ctx)
    assert.NoError(t, err)
    assert.Equal(t, "2024cur_sf9m1", match.Key)
    match, err = queue.PopMatch(ctx)
    assert.NoError(t, err)
    assert.Equal(t, "2024cur_sf12m1", match.Key)
    match, err = queue.PopMatch(ctx)
    assert.NoError(t, err)
    assert.Equal(t, "2024cur_f1m1", match.Key)
    match, err = queue.PopMatch(ctx)
    assert.NoError(t, err)
    assert.Equal(t, "2024cur_f1m2", match.Key)
}
