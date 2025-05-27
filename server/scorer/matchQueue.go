package scorer

import (
	"server/swagger"
	"sync"
)

type MatchQueue struct {
    lock sync.Mutex
    notEmpty *sync.Cond
    queue []swagger.Match
}

func NewMatchQueue() *MatchQueue {
    mq := &MatchQueue{}
    mq.notEmpty = sync.NewCond(&mq.lock)
    return mq
}

func (q *MatchQueue) PushMatch(match swagger.Match) {
    q.lock.Lock()
    defer q.lock.Unlock()
    q.queue = append(q.queue, match)
    q.notEmpty.Signal()
}

func (q *MatchQueue) PopMatch() swagger.Match {
    q.lock.Lock()
    defer q.lock.Unlock()
    for len(q.queue) == 0 {
        q.notEmpty.Wait()
    }
    match := q.queue[0]
    q.queue = q.queue[1:]
    return match
}
