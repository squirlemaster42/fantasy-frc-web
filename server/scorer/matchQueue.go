package scorer

import (
	"container/heap"
	"log/slog"
	"server/swagger"
	"server/utils"
)

type MatchQueue struct {
    matches []*swagger.Match
    matchPushChan chan *swagger.Match
    matchPopChan chan *swagger.Match
}

func InitQueue(queue *MatchQueue) {
    queue.matchPopChan = make(chan *swagger.Match)
    queue.matchPushChan = make(chan *swagger.Match, 50)
    heap.Init(queue)
    queue.watchMatchQueue()
}

func (q *MatchQueue) PushMatch(match swagger.Match) {
    q.matchPushChan <- &match
}

func (q *MatchQueue) PopMatch() swagger.Match {
    match, _ := <- q.matchPopChan
    return *match
}

func (q *MatchQueue) watchMatchQueue() chan bool {
    slog.Info("Starting match queue watch")
    quit := make(chan bool)

    var currentMatch *swagger.Match
    var currentIn = q.matchPushChan
    var currentOut chan<- *swagger.Match

    defer close(q.matchPopChan)

    go func() {
        for {
            select {
            case match, ok := <- currentIn:
                if !ok {
                    //Input has been closed
                    currentIn = nil
                    if currentMatch == nil {
                        return
                    }
                    continue
                }

                if currentMatch != nil {
                    heap.Push(q, currentMatch)
                }

                heap.Push(q, match)

                currentOut = q.matchPopChan

                currentMatch = heap.Pop(q).(*swagger.Match)
            case currentOut <- currentMatch:
                if q.Len() > 0 {
                    currentMatch = heap.Pop(q).(*swagger.Match)
                } else {
                    if currentIn == nil {
                        return
                    }

                    currentMatch = nil
                    currentOut = nil
                }
            }
        }
    }()

    return quit
}

func (q MatchQueue) Len() int {
    return len(q.matches)
}

func (q MatchQueue) Less(i int, j int) bool {
    return utils.CompareMatchOrder(q.matches[i].Key, q.matches[j].Key)
}

func (q MatchQueue) Swap(i int, j int) {
    q.matches[i], q.matches[j] = q.matches[j], q.matches[i]
}

func (q *MatchQueue) Push(match any) {
    q.matches = append(q.matches, match.(*swagger.Match))
}

func (q *MatchQueue) Pop() any {
    old := q.matches
    n := len(old)
    x := old[n - 1]
    q.matches = old[0 : n - 1]
    return x
}
