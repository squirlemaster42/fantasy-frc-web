package scorer

import (
	"context"
	"server/swagger"
)

type MatchQueue struct {
	pushCh chan swagger.Match
	popCh  chan swagger.Match
}

func NewMatchQueue() *MatchQueue {
	q := &MatchQueue{
		pushCh: make(chan swagger.Match),
		popCh:  make(chan swagger.Match),
	}
	go q.loop()
	return q
}

func (q *MatchQueue) loop() {
	var queue []swagger.Match
	for {
		var popCh chan<- swagger.Match
		var nextMatch swagger.Match
		if len(queue) > 0 {
			popCh = q.popCh
			nextMatch = queue[0]
		}

		select {
		case match := <-q.pushCh:
			queue = append(queue, match)
		case popCh <- nextMatch:
			queue = queue[1:]
		}
	}
}

func (q *MatchQueue) PushMatch(match swagger.Match) {
	q.pushCh <- match
}

func (q *MatchQueue) PopMatch(ctx context.Context) (swagger.Match, error) {
	select {
	case match := <-q.popCh:
		return match, nil
	case <-ctx.Done():
		return swagger.Match{}, ctx.Err()
	}
}
