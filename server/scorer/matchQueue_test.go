package scorer

import (
	"server/swagger"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchQueueOrdering(t *testing.T) {
    queue := &MatchQueue{}
    InitQueue(queue)
    queue.PushMatch(swagger.Match{
        Key: "2024cur_f1m2",
    })
    queue.PushMatch(swagger.Match{
        Key: "2024cur_f1m1",
    })
    queue.PushMatch(swagger.Match{
        Key: "2024cur_qm112",
    })
    queue.PushMatch(swagger.Match{
        Key: "2024cur_sf9m1",
    })
    queue.PushMatch(swagger.Match{
        Key: "2024cur_qm72",
    })
    queue.PushMatch(swagger.Match{
        Key: "2024cur_qm1",
    })
    queue.PushMatch(swagger.Match{
        Key: "2024cur_sf12m1",
    })
    queue.PushMatch(swagger.Match{
        Key: "2024cur_sf2m1",
    })
    assert.Equal(t, "2024cur_qm1", queue.PopMatch().Key)
    assert.Equal(t, "2024cur_qm72", queue.PopMatch().Key)
    assert.Equal(t, "2024cur_qm112", queue.PopMatch().Key)
    assert.Equal(t, "2024cur_sf2m1", queue.PopMatch().Key)
    assert.Equal(t, "2024cur_sf9m1", queue.PopMatch().Key)
    assert.Equal(t, "2024cur_sf12m1", queue.PopMatch().Key)
    assert.Equal(t, "2024cur_f1m1", queue.PopMatch().Key)
    assert.Equal(t, "2024cur_f1m2", queue.PopMatch().Key)
}
