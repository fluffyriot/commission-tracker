package worker

import (
	"context"
	"log"
	"time"

	"github.com/fluffyriot/commission-tracker/internal/database"
	"github.com/fluffyriot/commission-tracker/internal/fetcher"
	"github.com/fluffyriot/commission-tracker/internal/puller"
	"github.com/google/uuid"
)

type Worker struct {
	DB         *database.Queries
	Fetcher    *fetcher.Client
	Puller     *puller.Client
	InstVer    string
	EncryptKey []byte
	Ticker     *time.Ticker
	StopChan   chan bool
}

func NewWorker(db *database.Queries, fetcher *fetcher.Client, puller *puller.Client, instVer string, key []byte) *Worker {
	return &Worker{
		DB:         db,
		Fetcher:    fetcher,
		Puller:     puller,
		InstVer:    instVer,
		EncryptKey: key,
		StopChan:   make(chan bool),
	}
}

func (w *Worker) Start(interval time.Duration) {
	w.Ticker = time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-w.Ticker.C:
				w.SyncAll()
			case <-w.StopChan:
				w.Ticker.Stop()
				return
			}
		}
	}()
	log.Printf("Background worker started with interval: %v", interval)
}

func (w *Worker) Stop() {
	w.StopChan <- true
	log.Println("Background worker stopped")
}

func (w *Worker) SyncAll() {
	log.Println("Worker: Starting scheduled sync...")
	ctx := context.Background()

	users, err := w.DB.GetAllUsers(ctx)
	if err != nil {
		log.Printf("Worker Error getting users: %v", err)
		return
	}

	count := 0
	for _, user := range users {
		sources, err := w.DB.GetUserActiveSources(ctx, user.ID)
		if err != nil {
			log.Printf("Worker Error getting sources for user %s: %v", user.Username, err)
			continue
		}

		for _, source := range sources {

			go func(sid uuid.UUID) {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Worker Panic in sync: %v", r)
					}
				}()
				fetcher.SyncBySource(sid, w.DB, w.Fetcher, w.InstVer, w.EncryptKey)
			}(source.ID)
			count++
		}
	}
	log.Printf("Worker: Triggered sync for %d sources", count)
}
