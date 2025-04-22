package picking

import "sync"

type PickManager struct {
    draftId int
    lock sync.Mutex
    listeners []*PickListener
}

type PickEvent struct {
    pick string
}

type PickListener interface {
    recievePickEvent(pickEvent PickEvent)
}

//TODO Do we want to wrap this in another layer so you
//cannot create an more than one pick manager per draft
func NewPickManager(draftId int) *PickManager {
    return &PickManager{
        draftId: draftId,
    }
}

//Return error if pick is not able to be made
func (p *PickManager) makePick(pick string) error {
    p.lock.Lock()
    defer p.lock.Unlock()
    //validate pick
    for _, listener := range p.listeners {
        (*listener).recievePickEvent(PickEvent{
            pick: pick,
        })
    }
    return nil
}

func (p *PickManager) AddListener(listener PickListener) {

}

//TODO What should this take in?
func (p *PickManager) RemoveListener() {

}
