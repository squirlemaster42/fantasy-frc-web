package types

type PageData struct {
	DraftId int
	IsOwner bool
}

func NewPageData(draftId int, isOwner bool) *PageData {
	return &PageData{DraftId: draftId, IsOwner: isOwner}
}
