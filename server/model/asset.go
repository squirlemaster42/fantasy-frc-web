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

func loadDraftProfileAsset(database *sql.DB, draftId int) *Asset {
    return nil
}

//The actual html will need to base64 encode the image

func uploadDraftProfileAsset(database *sql.DB, draftId int, image image.Image) error {
    //Generate a unique id for the image
    //assetId := uuid.New()

    //Save the relative path to the database

    //Point the draft asset to this

    return nil
}

func loadUserProfileAsset(database *sql.DB, userUuid uuid.UUID) *Asset {
    return nil
}

func uploadUserProfileAsset(database *sql.DB, userUuid uuid.UUID, image image.Image) error {
    return nil
}
