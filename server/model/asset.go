package model

import (
	"database/sql"
	"image"

	"github.com/google/uuid"
)

type Asset struct {
    Id int
    Path string
    Asset image.Image //TODO Is this right
}

//TODO we should have something in the .env file to define the asset path
type AssetManager struct {
    AssetPath string
}

func (a *AssetManager) LoadDraftProfileAsset(database *sql.DB, draftId int) *Asset {
    return nil
}

//The actual html will need to base64 encode the image

//TODO Think about what sort of validation we need to do for replacing images (we might not need to do any since its obvious to the user)
func (a *AssetManager) UploadDraftProfileAsset(database *sql.DB, draftId int, image image.Image) error {
    //Generate a unique id for the image
    //assetId := uuid.New()

    //Save the relative path to the database

    //TODO We need to figure out the image type

    //Point the draft asset to this

    return nil
}

func (a *AssetManager) LoadUserProfileAsset(database *sql.DB, userGuid uuid.UUID) *Asset {
    return nil
}

func (a *AssetManager) UploadUserProfileAsset(database *sql.DB, userGuid uuid.UUID, image image.Image) error {
    return nil
}
