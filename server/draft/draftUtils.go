package draft

import (
	"context"
	"server/log"
	"time"
)

func SkipCurrentPick(ctx context.Context, draftActor *DraftActor, draftId int, currentPickId int) bool {
	replyChan := make(chan Result)
	skipped := false
	message := Message {
		Content: SkipCurrentPickMessage {
			CurrentPickId: draftActor.GetDraftState().CurrentPick.Id,
		},
		Reply: replyChan,
	}
	err := draftActor.PostMessage(context.TODO(), message)
	if err != nil {
		log.Warn(ctx, "Failed to post skip message to draft actor", "Draft Id", draftId, "Error", err)
		return false
	}
	select {
	case result := <- message.Reply:
		if result.Error != nil || !result.Value.(bool) {
			log.Warn(ctx, "Skipping current pick in draft failed", "Draft Id", draftId, "Current Pick Id", draftActor.GetDraftState().CurrentPick.Id, result.Error)
			skipped = false
		} else {
			skipped = true
		}
	case <- time.After(5 * time.Second):
		log.Warn(ctx, "Skipping current pick in draft timed out", "Draft Id", draftId, "Current Pick Id", draftActor.GetDraftState().CurrentPick.Id)
		skipped = false
	}
	return skipped
}
