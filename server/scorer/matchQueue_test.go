package scorer

import (
	"server/swagger"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchQueueOrdering(t *testing.T) {
    queue := MatchQueue{}
    InitQueue(&queue)
    queue.Push(&swagger.Match{
        Key: "2024cur_f1m2",
    })
    queue.Push(&swagger.Match{
        Key: "2024cur_f1m1",
    })
    queue.Push(&swagger.Match{
        Key: "2024cur_qm112 ",
    })
    queue.Push(&swagger.Match{
        Key: "2024cur_sf9m1",
    })
    queue.Push(&swagger.Match{
        Key: "2024cur_qm72",
    })
    queue.Push(&swagger.Match{
        Key: "2024cur_qm1",
    })
    queue.Push(&swagger.Match{
        Key: "2024cur_sf12m1",
    })
    queue.Push(&swagger.Match{
        Key: "2024cur_sf2m1",
    })
    assert.Equal(t, "2024cur_qm1", queue.Pop().(*swagger.Match).Key)
    assert.Equal(t, "2024cur_qm72", queue.Pop().(*swagger.Match).Key)
    assert.Equal(t, "2024cur_qm112", queue.Pop().(*swagger.Match).Key)
    assert.Equal(t, "2024cur_sf2m1", queue.Pop().(*swagger.Match).Key)
    assert.Equal(t, "2024cur_sf9m1", queue.Pop().(*swagger.Match).Key)
    assert.Equal(t, "2024cur_sf12m1", queue.Pop().(*swagger.Match).Key)
    assert.Equal(t, "2024cur_f1m1", queue.Pop().(*swagger.Match).Key)
    assert.Equal(t, "2024cur_f1m2", queue.Pop().(*swagger.Match).Key)
}
