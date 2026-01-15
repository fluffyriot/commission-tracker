package handlers

import (
	"github.com/fluffyriot/commission-tracker/internal/database"
	"github.com/fluffyriot/commission-tracker/internal/fetcher"
	"golang.org/x/oauth2"
)

type Handler struct {
	DB               *database.Queries
	Fetcher          *fetcher.Client
	InstVer          string
	EncryptKey       []byte
	OauthStateString string
	FBConfig         *oauth2.Config
	DBInitErr        error
	KeyB64Err2       error
	InstVerErr       error
	KeyB64Err1       error
}

func NewHandler(db *database.Queries, client *fetcher.Client, instVer string, key []byte, oauthState string, fbConfig *oauth2.Config, dbInitErr error, keyErr1 error, keyErr2 error, instVerErr error) *Handler {
	return &Handler{
		DB:               db,
		Fetcher:          client,
		InstVer:          instVer,
		EncryptKey:       key,
		OauthStateString: oauthState,
		FBConfig:         fbConfig,
		DBInitErr:        dbInitErr,
		KeyB64Err1:       keyErr1,
		KeyB64Err2:       keyErr2,
		InstVerErr:       instVerErr,
	}
}
