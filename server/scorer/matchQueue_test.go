package scorer

import (
	"container/heap"
	"server/swagger"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchQueueOrdering(t *testing.T) {
    queue := &MatchQueue{}
    InitQueue(queue)
    heap.Push(queue, &swagger.Match{
        Key: "2024cur_f1m2",
    })
    heap.Push(queue, &swagger.Match{
        Key: "2024cur_f1m1",
    })
    heap.Push(queue, &swagger.Match{
        Key: "2024cur_qm112",
    })
    heap.Push(queue, &swagger.Match{
        Key: "2024cur_sf9m1",
    })
    heap.Push(queue, &swagger.Match{
        Key: "2024cur_qm72",
    })
    heap.Push(queue, &swagger.Match{
        Key: "2024cur_qm1",
    })
    heap.Push(queue, &swagger.Match{
        Key: "2024cur_sf12m1",
    })
    heap.Push(queue, &swagger.Match{
        Key: "2024cur_sf2m1",
    })
    assert.Equal(t, "2024cur_qm1", heap.Pop(queue).(*swagger.Match).Key)
    assert.Equal(t, "2024cur_qm72", heap.Pop(queue).(*swagger.Match).Key)
    assert.Equal(t, "2024cur_qm112", heap.Pop(queue).(*swagger.Match).Key)
    assert.Equal(t, "2024cur_sf2m1", heap.Pop(queue).(*swagger.Match).Key)
    assert.Equal(t, "2024cur_sf9m1", heap.Pop(queue).(*swagger.Match).Key)
    assert.Equal(t, "2024cur_sf12m1", heap.Pop(queue).(*swagger.Match).Key)
    assert.Equal(t, "2024cur_f1m1", heap.Pop(queue).(*swagger.Match).Key)
    assert.Equal(t, "2024cur_f1m2", heap.Pop(queue).(*swagger.Match).Key)
}
