package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fluffyriot/commission-tracker/cmd/exports"
	"github.com/fluffyriot/commission-tracker/cmd/fetcher"
	"github.com/fluffyriot/commission-tracker/internal/auth"
	"github.com/fluffyriot/commission-tracker/internal/config"
	"github.com/fluffyriot/commission-tracker/internal/database"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var (
	dbQueries  *database.Queries
	dbInitErr  error
	keyB64Err1 error
	keyB64Err2 error
	instVerErr error
)

func main() {

	httpsPort := os.Getenv("HTTPS_PORT")
	if httpsPort == ":" {
		log.Fatal("HTTPS_PORT is not set in the .env")
	}

	appPort := os.Getenv("APP_PORT")
	if appPort == ":" {
		log.Fatal("APP_PORT is not set in the .env")
	}

	clientIP := os.Getenv("LOCAL_IP")
	if clientIP == "" {
		log.Fatal("LOCAL_IP is not set in the .env")
	}

	instVer := os.Getenv("INSTAGRAM_API_VERSION")
	if instVer == "" {
		instVerErr = errors.New("INSTAGRAM_API_VERSION not set in .env")
	}

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

	dbQueries, dbInitErr = config.LoadDatabase()
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

	r.GET("/", rootHandler)

	r.GET("/auth/facebook/login", func(c *gin.Context) {

		sid, err := uuid.Parse(c.Query("sid"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source_id"})
			return
		}

		pid := c.Query("pid")
		if pid == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "profile_id is required"})
			return
		}

		payload := base64.URLEncoding.EncodeToString([]byte(sid.String() + ":" + pid))

		state := oauthStateString + "|" + payload

		url := fbConfig.AuthCodeURL(state)
		c.Redirect(http.StatusTemporaryRedirect, url)

	})

	r.GET("/auth/facebook/callback", func(c *gin.Context) {
		rawState := c.Query("state")
		parts := strings.SplitN(rawState, "|", 2)

		if len(parts) != 2 || parts[0] != oauthStateString {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid oauth state"})
			return
		}

		decoded, err := base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state payload"})
			return
		}

		values := strings.SplitN(string(decoded), ":", 2)
		if len(values) != 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state format"})
			return
		}

		sid, err := uuid.Parse(values[0])
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sid in state"})
			return
		}

		pid := values[1]

		code := c.Query("code")
		token, err := fbConfig.Exchange(c, code)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "token exchange failed", "details": err.Error()})
			return
		}

		longLivedToken, err := auth.ExchangeLongLivedToken(token.AccessToken, fbConfig)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "long-lived token exchange failed", "details": err.Error()})
			return
		}
		token.AccessToken = longLivedToken

		client := fbConfig.Client(c, token)
		resp, err := client.Get("https://graph.facebook.com/me?fields=id,email")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user info"})
			return
		}
		defer resp.Body.Close()

		tokenStr, err := auth.OauthTokenToString(token)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to serialize token", "details": err.Error()})
			return
		}

		err = auth.InsertToken(dbQueries, sid, tokenStr, pid, encryptKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store token", "details": err.Error()})
			return
		}

		c.Redirect(http.StatusSeeOther, "/")
	})

	/* 	r.GET("/refresh/:id", func(c *gin.Context) {
		sid, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source id"})
			return
		}

		token, tokenId, err := auth.GetToken(context.Background(), dbQueries, encryptKey, sid)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
			return
		}

		newToken, err := auth.ExchangeLongLivedToken(token, fbConfig)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to refresh token", "details": err.Error()})
			return
		}

		err = dbQueries.DeleteTokenById(context.Background(), tokenId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete old token", "details": err.Error()})
			return
		}

		err = auth.InsertToken(dbQueries, sid, newToken, encryptKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert new token", "details": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "token refreshed", "access_token": newToken})

	}) */

	r.GET("/exports", exportsHandler)
	r.POST("/export/start", exportStartHandler(dbQueries))
	r.POST("/exports/deleteAll", exportDeleteAllHandler(dbQueries))

	r.GET("/outputs/*filepath", func(c *gin.Context) {
		p := c.Param("filepath")[1:]
		c.FileAttachment(filepath.Join("./outputs", p), filepath.Base(p))
	})
	r.POST("/user/setup", userSetupHandler)
	r.POST("/sources/setup", sourcesSetupHandler(encryptKey))
	r.POST("/sources/deactivate", deactivateSourceHandler)
	r.POST("/sources/activate", activateSourceHandler)
	r.POST("/sources/delete", deleteSourceHandler)
	r.POST("/sources/sync", syncSourceHandler(encryptKey, dbQueries, client, instVer))
	r.POST("/sources/syncAll", syncAllHandler(encryptKey, dbQueries, client, instVer))

	r.POST("/reset", func(c *gin.Context) {
		err := dbQueries.EmptyUsers(context.Background())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	if err := r.Run(":" + appPort); err != nil {
		log.Fatal(err)
	}
}

func rootHandler(c *gin.Context) {

	if dbInitErr != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": dbInitErr.Error(),
		})
		return
	}

	if keyB64Err1 != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": keyB64Err1.Error(),
		})
		return
	}

	if keyB64Err2 != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": keyB64Err2.Error(),
		})
		return
	}

	if instVerErr != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": instVerErr.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	users, err := dbQueries.GetAllUsers(ctx)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": err.Error(),
		})
		return
	}

	if len(users) == 0 {
		c.HTML(http.StatusOK, "user-setup.html", nil)
		return
	}

	user := users[0]

	sources, err := dbQueries.GetUserSources(ctx, user.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": err.Error(),
		})
		return
	}
	c.HTML(http.StatusOK, "index.html", gin.H{
		"username": user.Username,
		"user_id":  user.ID,
		"sources":  sources,
	})
}

func exportsHandler(c *gin.Context) {

	if dbInitErr != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": dbInitErr.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	users, err := dbQueries.GetAllUsers(ctx)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": err.Error(),
		})
		return
	}

	if len(users) == 0 {
		c.HTML(http.StatusOK, "user-setup.html", nil)
		return
	}

	user := users[0]

	exports, err := dbQueries.GetLast20ExportsByUserId(ctx, user.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": err.Error(),
		})
		return
	}
	c.HTML(http.StatusOK, "exports.html", gin.H{
		"username":    user.Username,
		"user_id":     user.ID,
		"sync_method": user.SyncMethod,
		"exports":     exports,
	})
}

func userSetupHandler(c *gin.Context) {
	username := c.PostForm("username")
	if username == "" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": "username is required",
		})
		return
	}

	syncMethod := c.PostForm("sync_method")
	if syncMethod == "" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": "Sync method is required",
		})
		return
	}

	_, _, err := config.CreateUserFromForm(dbQueries, username, syncMethod)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Redirect(http.StatusSeeOther, "/")
}

func exportStartHandler(dbQueries *database.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId, err := uuid.Parse(c.PostForm("user_id"))
		if err != nil {
			c.HTML(http.StatusBadRequest, "error.html", gin.H{
				"error": err.Error(),
			})
			return
		}

		syncMethod := c.PostForm("sync_method")
		if syncMethod == "" {
			c.HTML(http.StatusBadRequest, "error.html", gin.H{
				"error": "Sync method is required",
			})
			return
		}

		go func(uid uuid.UUID, method string) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in background sync: %v", r)
				}
			}()
			exports.InitiateExport(uid, method, dbQueries)
		}(userId, syncMethod)

		c.Redirect(http.StatusSeeOther, "/")
	}
}

func exportDeleteAllHandler(dbQueries *database.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId, err := uuid.Parse(c.PostForm("user_id"))
		if err != nil {
			c.HTML(http.StatusBadRequest, "error.html", gin.H{
				"error": err.Error(),
			})
			return
		}

		go func(uid uuid.UUID) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in background sync: %v", r)
				}
			}()
			exports.DeleteAllExports(uid, dbQueries)
		}(userId)

		c.Redirect(http.StatusSeeOther, "/")
	}
}

func sourcesSetupHandler(encryptKey []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.PostForm("user_id")
		network := c.PostForm("network")
		username := c.PostForm("username")
		profile_id := c.PostForm("instagram_profile_id")

		if userID == "" || network == "" || username == "" {
			c.HTML(http.StatusBadRequest, "error.html", gin.H{
				"error": "all fields are required",
			})
			return
		}

		sid, _, err := config.CreateSourceFromForm(
			dbQueries,
			userID,
			network,
			username,
		)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"error": err.Error(),
			})
			return
		}

		if network == "Instagram" {
			c.Redirect(http.StatusSeeOther, "/auth/facebook/login?sid="+sid+"&pid="+profile_id)
			return
		}

		c.Redirect(http.StatusSeeOther, "/")
	}
}

func deactivateSourceHandler(c *gin.Context) {
	sourceID, err := uuid.Parse(c.PostForm("source_id"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": err.Error(),
		})
		return
	}

	_, err = dbQueries.ChangeSourceStatusById(
		context.Background(),
		database.ChangeSourceStatusByIdParams{
			ID:           sourceID,
			IsActive:     false,
			SyncStatus:   "Deactivated",
			StatusReason: sql.NullString{String: "Sync stopped by the user", Valid: true},
		},
	)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Redirect(http.StatusSeeOther, "/")
}

func deleteSourceHandler(c *gin.Context) {
	sourceID, err := uuid.Parse(c.PostForm("source_id"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": err.Error(),
		})
		return
	}

	err = dbQueries.DeleteSource(context.Background(), sourceID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Redirect(http.StatusSeeOther, "/")
}

func activateSourceHandler(c *gin.Context) {
	sourceID, err := uuid.Parse(c.PostForm("source_id"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": err.Error(),
		})
		return
	}

	_, err = dbQueries.ChangeSourceStatusById(
		context.Background(),
		database.ChangeSourceStatusByIdParams{
			ID:           sourceID,
			IsActive:     true,
			SyncStatus:   "Initialized",
			StatusReason: sql.NullString{},
		},
	)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Redirect(http.StatusSeeOther, "/")
}

func syncSourceHandler(encryptKey []byte, dbQueries *database.Queries, client *fetcher.Client, ver string) gin.HandlerFunc {
	return func(c *gin.Context) {
		sourceID, err := uuid.Parse(c.PostForm("source_id"))
		if err != nil {
			c.HTML(http.StatusBadRequest, "error.html", gin.H{
				"error": err.Error(),
			})
			return
		}

		go func(sid uuid.UUID) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in background sync: %v", r)
				}
			}()
			fetcher.SyncBySource(sid, dbQueries, client, ver, encryptKey)
		}(sourceID)

		c.Redirect(http.StatusSeeOther, "/")
	}
}

func syncAllHandler(encryptKey []byte, dbQueries *database.Queries, client *fetcher.Client, ver string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := uuid.Parse(c.PostForm("user_id"))
		if err != nil {
			c.HTML(http.StatusBadRequest, "error.html", gin.H{
				"error": err.Error(),
			})
			return
		}

		sources, err := dbQueries.GetUserActiveSources(context.Background(), userID)

		for _, sourceID := range sources {
			go func(sid uuid.UUID) {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("panic in background sync: %v", r)
					}
				}()
				fetcher.SyncBySource(sid, dbQueries, client, ver, encryptKey)
			}(sourceID.ID)
		}

		c.Redirect(http.StatusSeeOther, "/")
	}
}
