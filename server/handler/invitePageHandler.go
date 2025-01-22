package handler

import "github.com/labstack/echo/v4"

import (
	draftView "server/view/draft"
)

func (h *Handler) HandleViewInvites(c echo.Context) error {
    inviteIndex := draftView.DraftInviteIndex()
    //TODO Get username
    inviteView := draftView.DraftInvite(" | Draft Invites", true, "", inviteIndex)
    err := Render(c, inviteView)
    return err
}
