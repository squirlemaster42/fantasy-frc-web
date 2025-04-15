package scorer

import (
	"container/heap"
	"server/assert"
	"server/swagger"
	"strconv"
	"strings"
)

type MatchQueue []*swagger.Match

//TODO Something is broken here
func InitQueue(queue *MatchQueue) {
    heap.Init(queue)
}

func (q MatchQueue) Len() int {
    return len(q)
}

func (q MatchQueue) Less(i int, j int) bool {
    return compareMatchOrder(q[i].Key, q[j].Key)
}

func (q MatchQueue) Swap(i int, j int) {
    q[i], q[j] = q[j], q[i]
}

func (q *MatchQueue) Push(match any) {
    *q = append(*q, match.(*swagger.Match))
}

func (q *MatchQueue) Pop() any {
    old := *q
    n := len(old)
    x := old[n - 1]
    *q = old[0 : n - 1]
    return x
}

// Return true if matchA comes before matchB
func compareMatchOrder(matchA string, matchB string) bool {
    assert := assert.CreateAssertWithContext("Compare Match Order")
    assert.AddContext("Match A", matchA)
    assert.AddContext("Match B", matchB)
    matchALevel := getMatchLevel(matchA)
    matchBLevel := getMatchLevel(matchB)
    assert.AddContext("Match A Level", matchALevel)
    assert.AddContext("Match B Level", matchBLevel)
    aPrecidence, ok := matchPrecidence()[matchALevel]
    assert.RunAssert(ok, "Match Precidence Was Not Found")
    bPrecidence, ok := matchPrecidence()[matchBLevel]
    assert.RunAssert(ok, "Match Precidence Was Not Found")

    if aPrecidence != bPrecidence {
        return aPrecidence < bPrecidence
    }

    assert.RunAssert(matchALevel == matchBLevel, "Match levels are not the same")

    if matchALevel == "qm" {
        splitMatchA := strings.Split(matchA, "_")
        splitMatchB := strings.Split(matchB, "_")
        assert.RunAssert(len(splitMatchA) == 2, "Match A string was invalid")
        assert.RunAssert(len(splitMatchB) == 2, "Match B string was invalid")
        matchANumStr := strings.TrimSpace(splitMatchA[1][2:])
        matchBNumStr := strings.TrimSpace(splitMatchB[1][2:])
        assert.AddContext("Match A Num", matchANumStr)
        assert.AddContext("Match B Num", matchBNumStr)
        matchANum, err := strconv.Atoi(matchANumStr)
        assert.NoError(err, "Match A num Atoi failed")
        matchBNum, err := strconv.Atoi(matchBNumStr)
        assert.NoError(err, "Match B num Atoi failed")
        return matchANum < matchBNum
    }

    if matchALevel == "f" {
        splitMatchA := strings.Split(matchA, "_")
        splitMatchB := strings.Split(matchB, "_")
        assert.RunAssert(len(splitMatchA) == 2, "Match A string was invalid")
        assert.RunAssert(len(splitMatchB) == 2, "Match B string was invalid")
        splitMatchA = strings.Split(splitMatchA[1][1:], "m")
        splitMatchB = strings.Split(splitMatchB[1][1:], "m")
        assert.RunAssert(len(splitMatchA) == 2, "Match A string was invalid")
        assert.RunAssert(len(splitMatchB) == 2, "Match B string was invalid")
        matchANum, err := strconv.Atoi(splitMatchA[0])
        assert.NoError(err, "Match A num Atoi failed")
        matchBNum, err := strconv.Atoi(splitMatchB[0])
        assert.NoError(err, "Match B num Atoi failed")

        if matchANum != matchBNum {
            return matchANum < matchBNum
        }

        assert.RunAssert(matchANum == matchBNum, "Match nums are the same but shouldn't be")

        matchANum, err = strconv.Atoi(splitMatchA[1])
        assert.NoError(err, "Match A num Atoi failed")
        matchBNum, err = strconv.Atoi(splitMatchB[1])
        assert.NoError(err, "Match B num Atoi failed")

        return matchANum < matchBNum
    }

    if matchALevel == "sf" {
        splitMatchA := strings.Split(matchA, "_")
        splitMatchB := strings.Split(matchB, "_")
        assert.RunAssert(len(splitMatchA) == 2, "Match A string was invalid")
        assert.RunAssert(len(splitMatchB) == 2, "Match B string was invalid")
        splitMatchA = strings.Split(splitMatchA[1][2:], "m")
        splitMatchB = strings.Split(splitMatchB[1][2:], "m")
        assert.RunAssert(len(splitMatchA) == 2, "Match A string was invalid")
        assert.RunAssert(len(splitMatchB) == 2, "Match B string was invalid")
        matchANum, err := strconv.Atoi(splitMatchA[0])
        assert.NoError(err, "Match A num Atoi failed")
        matchBNum, err := strconv.Atoi(splitMatchB[0])
        assert.NoError(err, "Match B num Atoi failed")

        if matchANum != matchBNum {
            return matchANum < matchBNum
        }

        assert.RunAssert(matchANum == matchBNum, "Match nums are the same but shouldn't be")

        matchANum, err = strconv.Atoi(splitMatchA[1])
        assert.NoError(err, "Match A num Atoi failed")
        matchBNum, err = strconv.Atoi(splitMatchB[1])
        assert.NoError(err, "Match B num Atoi failed")

        return matchANum < matchBNum
    }

    assert.RunAssert(1 == 0, "Unknown match type found")
    return false // This is unreachable
}
