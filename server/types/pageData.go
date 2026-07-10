package types

type PageData struct {
	DraftId int
	DraftName string
	IsOwner bool
}

func NewPageData(draftId int, draftName string, isOwner bool) *PageData {
	return &PageData{
		DraftId: draftId,
		DraftName: draftName,
		IsOwner: isOwner,
	}
}
