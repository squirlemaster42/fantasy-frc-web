package model

type Match struct {
    TbaId string
    Played bool
    RedScore int
    BlueScore int
}

func AddMatch(tbaId string) error {
    return nil
}

func AddMatchWithScore(tbaId string, redScore int, blueScore int) error {
    return nil
}

func UpdateScore(tbaId string, redScore int, blueScore int) error {
    return nil
}

func GetMatch(tbaId string) (error, Match) {
    return nil, Match{}
}
