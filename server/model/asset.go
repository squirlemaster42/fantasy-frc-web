package model

import (
	"database/sql"
	"image"

	"github.com/google/uuid"
)

type Asset struct {
    Id int
    Path string
    Asset image.Image
}

type AssetManager struct {
    AssetPath string
}

func (a *AssetManager) LoadDraftProfileAsset(database *sql.DB, draftId int) *Asset {
    return nil
}

//The actual html will need to base64 encode the image

func (a *AssetManager) UploadDraftProfileAsset(database *sql.DB, draftId int, image image.Image) error {
    //Generate a unique id for the image
    //assetId := uuid.New()

    //Save the relative path to the database

    //Point the draft asset to this

    return nil
}

func (a *AssetManager) LoadUserProfileAsset(database *sql.DB, userUuid uuid.UUID) *Asset {
    return nil
}

func (a *AssetManager) UploadUserProfileAsset(database *sql.DB, userUuid uuid.UUID, image image.Image) error {
    return nil
}
