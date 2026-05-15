package model

import (
	"context"
	"database/sql"
	"image"

	"github.com/google/uuid"
)

type SQLAssetStore struct {
	db *sql.DB
}

func NewSQLAssetStore(db *sql.DB) *SQLAssetStore {
	return &SQLAssetStore{db: db}
}

func (s *SQLAssetStore) LoadDraftProfileAsset(ctx context.Context, draftId int) (*Asset, error) {
	return loadDraftProfileAsset(s.db, draftId)
}

func (s *SQLAssetStore) UploadDraftProfileAsset(ctx context.Context, draftId int, image image.Image) error {
	return uploadDraftProfileAsset(s.db, draftId, image)
}

func (s *SQLAssetStore) LoadUserProfileAsset(ctx context.Context, userUuid uuid.UUID) (*Asset, error) {
	return loadUserProfileAsset(s.db, userUuid)
}

func (s *SQLAssetStore) UploadUserProfileAsset(ctx context.Context, userUuid uuid.UUID, image image.Image) error {
	return uploadUserProfileAsset(s.db, userUuid, image)
}
