package model

type Draft struct {
    Name string
    Players []struct {
        Name string
        Picks []string
    }
}
