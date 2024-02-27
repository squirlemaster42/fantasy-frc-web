package scoring

type Scorer struct {

}

func NewScorer(tbaToken string) *Scorer {
    scorer := Scorer{}
    return &scorer
}
