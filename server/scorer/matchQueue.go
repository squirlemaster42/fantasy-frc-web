package scorer

import (
	"container/heap"
	"log/slog"
	"server/assert"
	"server/swagger"
	"server/utils"
)

type matchQueueChanPopMsg struct {
    match chan swagger.Match
}


type matchQueueChanPushMsg struct {
    match swagger.Match
}

type MatchQueue struct {
    matches []swagger.Match
    matchPushChan chan matchQueueChanPushMsg
    matchPopChan chan matchQueueChanPopMsg
    quitChan chan bool
}

func InitQueue(queue *MatchQueue) {
    queue.matchPopChan = make(chan matchQueueChanPopMsg)
    queue.matchPushChan = make(chan matchQueueChanPushMsg)
    heap.Init(queue)
    queue.quitChan = queue.watchMatchQueue()
}

func (q *MatchQueue) PushMatch(match swagger.Match) {
    q.matchPushChan <- matchQueueChanPushMsg {
        match: match,
    }
}

func (q *MatchQueue) PopMatch() swagger.Match {
    match := make(chan swagger.Match)
    q.matchPopChan <- matchQueueChanPopMsg{
        match: match,
    }
    return <- match
}

func (q *MatchQueue) StopMatchQueue() {
    q.quitChan <- true
}

func (q *MatchQueue) watchMatchQueue() chan bool {
    slog.Info("Starting match queue watch")
    quit := make(chan bool)

    go func() {
        for {
            assert := assert.CreateAssertWithContext("Watch Match Queue")
            select {
            case <- quit:
                return
            case popMatch := <- q.matchPopChan:
                match, ok := heap.Pop(q).(swagger.Match)
                assert.AddContext("Match", match)
                assert.RunAssert(ok, "Something got into the queue that was not a match")
                popMatch.match <- match
            case pushMatch := <- q.matchPushChan:
                heap.Push(q, pushMatch.match)
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
    q.matches = append(q.matches, match.(swagger.Match))
}

func (q *MatchQueue) Pop() any {
    old := q.matches
    n := len(old)
    x := old[n - 1]
    q.matches = old[0 : n - 1]
    return x
}
