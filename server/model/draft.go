package model

import "time"

type Draft struct {
    Id int
    DisplayName string
    Owner int //User
}

type Pick struct {
    Id int
    Player int //User
    PickOrder int
    Pick string //Team
    PickTime time.Time
}

type DraftInvite struct {
    Id int
    draftId int //Draft
    invitedPlayer int //User
    invitingPlayer int //User
    sentTime time.Time
    acceptedTime time.Time
    accepted bool
}

func CreateDraft(owner int, displayName string) error {
    return nil
}

func InvitePlayer(draft int, invitedPlayer int) error {
    return nil
}

func AcceptInvite(draft int, player int) error {
    return nil
}

func GetInvites(player int) (error, []int) {
    return nil, []int{}
}

func GetPicks(draft int) (error, []Pick) {
    return nil, []Pick{}
}

func GetNextPick(draft int) (error, Pick) {
    return nil, Pick{}
}

func MakePick(pick Pick) error {
    return nil
}
