package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fluffyriot/commission-tracker/internal/api/handlers"
	"github.com/fluffyriot/commission-tracker/internal/auth"
	"github.com/fluffyriot/commission-tracker/internal/config"
	"github.com/fluffyriot/commission-tracker/internal/fetcher"
	"github.com/fluffyriot/commission-tracker/internal/worker"
	"github.com/gin-gonic/gin"
)

func main() {

	httpsPort := os.Getenv("HTTPS_PORT")
	if httpsPort == "" {
		log.Fatal("HTTPS_PORT is not set in the .env")
	}

	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		log.Fatal("APP_PORT is not set in the .env")
	}

	clientIP := os.Getenv("LOCAL_IP")
	if clientIP == "" {
		log.Fatal("LOCAL_IP is not set in the .env")
	}

	var instVerErr error
	instVer := os.Getenv("INSTAGRAM_API_VERSION")
	if instVer == "" {
		instVerErr = errors.New("INSTAGRAM_API_VERSION not set in .env")
	}

	var keyB64Err1 error
	keyB64 := os.Getenv("TOKEN_ENCRYPTION_KEY")
	if keyB64 == "" {
		keyB64Err1 = errors.New("TOKEN_ENCRYPTION_KEY not set in .env")
	}

	encryptKey, keyB64Err2 := base64.StdEncoding.DecodeString(keyB64)
	if keyB64Err2 != nil || len(encryptKey) != 32 {
		keyB64Err2 = fmt.Errorf("Error encoding encryption key: %v", keyB64Err2)
	}

	client := fetcher.NewClient(600 * time.Second)

	r := gin.Default()

	r.SetTrustedProxies(nil)

	r.Static("/static", "./static")

	r.LoadHTMLGlob("templates/*.html")

	dbQueries, dbInitErr := config.LoadDatabase()
	if dbInitErr != nil {
		log.Printf("database init failed: %v", dbInitErr)
	}

	oauthStateString := os.Getenv("OAUTH_ENCRYPTION_KEY")
	fbConfig := auth.GenerateFacebookConfig(
		os.Getenv("FACEBOOK_APP_ID"),
		os.Getenv("FACEBOOK_APP_SECRET"),
		clientIP,
		httpsPort,
	)

	// Start background worker
	w := worker.NewWorker(dbQueries, client, instVer, encryptKey)
	w.Start(1 * time.Hour) // Sync every hour

	h := handlers.NewHandler(
		dbQueries,
		client,
		instVer,
		encryptKey,
		oauthStateString,
		fbConfig,
		dbInitErr,
		keyB64Err1,
		keyB64Err2,
		instVerErr,
	)

	r.GET("/", h.RootHandler)

	r.GET("/auth/facebook/login", h.FacebookLoginHandler)
	r.GET("/auth/facebook/callback", h.FacebookCallbackHandler)

	r.GET("/exports", h.ExportsHandler)
	r.POST("/export/start", h.ExportStartHandler)
	r.POST("/exports/deleteAll", h.ExportDeleteAllHandler)

	r.GET("/outputs/*filepath", h.DownloadExportHandler)

	r.POST("/user/setup", h.UserSetupHandler)
	r.POST("/sources/setup", h.SourcesSetupHandler)
	r.POST("/sources/deactivate", h.DeactivateSourceHandler)
	r.POST("/sources/activate", h.ActivateSourceHandler)
	r.POST("/sources/delete", h.DeleteSourceHandler)
	r.POST("/sources/sync", h.SyncSourceHandler)
	r.POST("/sources/syncAll", h.SyncAllHandler)

	r.POST("/reset", h.ResetHandler)

	r.GET("/stats", h.StatsHandler)
	r.GET("/analytics", h.AnalyticsPageHandler)

	if err := r.Run(":" + appPort); err != nil {
		log.Fatal(err)
	}
}
