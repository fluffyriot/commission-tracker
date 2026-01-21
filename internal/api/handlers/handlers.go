package handlers

import (
	"github.com/fluffyriot/commission-tracker/internal/config"
	"github.com/fluffyriot/commission-tracker/internal/database"
	"github.com/fluffyriot/commission-tracker/internal/fetcher"
	"github.com/fluffyriot/commission-tracker/internal/puller"
	"github.com/fluffyriot/commission-tracker/internal/worker"
)

type Handler struct {
	DB      *database.Queries
	Fetcher *fetcher.Client
	Puller  *puller.Client
	Config  *config.AppConfig
	Worker  *worker.Worker
}

func NewHandler(db *database.Queries, clientFetch *fetcher.Client, clientPull *puller.Client, cfg *config.AppConfig, w *worker.Worker) *Handler {
	return &Handler{
		DB:      db,
		Fetcher: clientFetch,
		Puller:  clientPull,
		Config:  cfg,
		Worker:  w,
	}
}
