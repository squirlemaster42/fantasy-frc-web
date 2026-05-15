package model

import (
	"context"
	"image"

	"github.com/google/uuid"
)

type AssetStore interface {
	LoadDraftProfileAsset(ctx context.Context, draftId int) (*Asset, error)
	UploadDraftProfileAsset(ctx context.Context, draftId int, image image.Image) error
	LoadUserProfileAsset(ctx context.Context, userUuid uuid.UUID) (*Asset, error)
	UploadUserProfileAsset(ctx context.Context, userUuid uuid.UUID, image image.Image) error
}
