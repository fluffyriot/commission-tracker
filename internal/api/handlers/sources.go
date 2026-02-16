// SPDX-License-Identifier: AGPL-3.0-only
package handlers

import (
	"context"
	"database/sql"
	"log"
	"net/http"

	"github.com/fluffyriot/rpsync/internal/config"
	"github.com/fluffyriot/rpsync/internal/database"
	"github.com/fluffyriot/rpsync/internal/helpers"
	"github.com/fluffyriot/rpsync/internal/pusher"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) SourcesHandler(c *gin.Context) {
	if h.Config.DBInitErr != nil {
		c.HTML(http.StatusInternalServerError, "error.html", h.CommonData(c, gin.H{
			"error": h.Config.DBInitErr.Error(),
			"title": "Error",
		}))
		return
	}

	ctx := c.Request.Context()

	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	sources, err := h.DB.GetUserSources(ctx, user.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", h.CommonData(c, gin.H{
			"error": err.Error(),
			"title": "Error",
		}))
		return
	}

	c.HTML(http.StatusOK, "sources.html", h.CommonData(c, gin.H{
		"username":          user.Username,
		"user_id":           user.ID,
		"sources":           sources,
		"available_sources": helpers.AvailableSources,
		"title":             "Sources",
	}))
}

func (h *Handler) HandleGetSourcesAPI(c *gin.Context) {
	if h.Config.DBInitErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": h.Config.DBInitErr.Error()})
		return
	}

	ctx := c.Request.Context()

	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	sources, err := h.DB.GetUserSources(ctx, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sources)
}

func (h *Handler) SourcesSetupHandler(c *gin.Context) {
	userID := c.PostForm("user_id")
	network := c.PostForm("network")
	username := c.PostForm("username")
	field1 := c.PostForm("field_1")
	field2 := c.PostForm("field_2")
	field3 := c.PostForm("field_3")
	field4 := c.PostForm("field_4")
	fieldLong := c.PostForm("field_long")

	if userID == "" || network == "" || username == "" {
		c.HTML(http.StatusBadRequest, "error.html", h.CommonData(c, gin.H{
			"error": "All fields are required",
			"title": "Error",
		}))
		return
	}

	params := config.SourceCreationParams{
		UserID:        userID,
		Network:       network,
		Username:      username,
		Field1:        field1,
		Field2:        field2,
		Field3:        field3,
		Field4:        field4,
		FieldLong:     fieldLong,
		EncryptionKey: h.Config.TokenEncryptionKey,
	}

	sid, _, err := config.CreateSourceFromForm(h.DB, params)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", h.CommonData(c, gin.H{
			"error": err.Error(),
			"title": "Error",
		}))
		return
	}

	if network == "Instagram" {
		session := sessions.Default(c)
		session.Set("app_id_"+sid, field2)
		session.Set("app_secret_"+sid, field3)
		session.Save()

		c.Redirect(http.StatusSeeOther, "/auth/facebook/login?sid="+sid+"&pid="+field1)
		return
	}

	if network == "TikTok" {
		c.Redirect(http.StatusSeeOther, "/auth/tiktok/login?username="+username)
		return
	}

	c.Redirect(http.StatusSeeOther, "/sources")
}

func (h *Handler) DeactivateSourceHandler(c *gin.Context) {
	sourceID, err := uuid.Parse(c.PostForm("source_id"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", h.CommonData(c, gin.H{
			"error": err.Error(),
			"title": "Error",
		}))
		return
	}

	_, err = h.DB.ChangeSourceStatusById(
		context.Background(),
		database.ChangeSourceStatusByIdParams{
			ID:           sourceID,
			IsActive:     false,
			SyncStatus:   "Deactivated",
			StatusReason: sql.NullString{String: "Sync stopped by the user", Valid: true},
		},
	)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", h.CommonData(c, gin.H{
			"error": err.Error(),
			"title": "Error",
		}))
		return
	}

	c.Redirect(http.StatusSeeOther, "/sources")
}

func (h *Handler) ActivateSourceHandler(c *gin.Context) {
	sourceID, err := uuid.Parse(c.PostForm("source_id"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", h.CommonData(c, gin.H{
			"error": err.Error(),
			"title": "Error",
		}))
		return
	}

	_, err = h.DB.ChangeSourceStatusById(
		context.Background(),
		database.ChangeSourceStatusByIdParams{
			ID:           sourceID,
			IsActive:     true,
			SyncStatus:   "Initialized",
			StatusReason: sql.NullString{},
		},
	)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", h.CommonData(c, gin.H{
			"error": err.Error(),
			"title": "Error",
		}))
		return
	}

	c.Redirect(http.StatusSeeOther, "/sources")
}

func (h *Handler) DeleteSourceHandler(c *gin.Context) {
	sourceID, err := uuid.Parse(c.PostForm("source_id"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", h.CommonData(c, gin.H{
			"error": err.Error(),
			"title": "Error",
		}))
		return
	}

	syncedTargets, err := h.DB.GetSourcesOfTarget(context.Background(), sourceID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", h.CommonData(c, gin.H{
			"error": err.Error(),
			"title": "Error",
		}))
		return
	}

	for _, target := range syncedTargets {
		err = pusher.RemoveByTarget(target.TargetID, sourceID, h.DB, h.Puller, h.Config.TokenEncryptionKey)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", h.CommonData(c, gin.H{
				"error": err.Error(),
				"title": "Error",
			}))
			return
		}
	}

	err = h.DB.DeleteSource(context.Background(), sourceID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", h.CommonData(c, gin.H{
			"error": err.Error(),
			"title": "Error",
		}))
		return
	}

	c.Redirect(http.StatusSeeOther, "/sources")
}

func (h *Handler) SyncSourceHandler(c *gin.Context) {
	sourceID, err := uuid.Parse(c.PostForm("source_id"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", h.CommonData(c, gin.H{
			"error": err.Error(),
			"title": "Error",
		}))
		return
	}

	go func(sid uuid.UUID) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in background sync: %v", r)
			}
		}()
		h.Worker.SyncSource(sid)
	}(sourceID)

	c.Redirect(http.StatusSeeOther, "/sources")
}

func (h *Handler) HandleExportCookies(c *gin.Context) {
	sourceID, err := uuid.Parse(c.Query("source_id"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", h.CommonData(c, gin.H{
			"error": "Invalid source ID",
			"title": "Error",
		}))
		return
	}

	source, err := h.DB.GetSourceById(context.Background(), sourceID)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", h.CommonData(c, gin.H{
			"error": "Source not found",
			"title": "Error",
		}))
		return
	}

	if source.Network != "TikTok" {
		c.HTML(http.StatusBadRequest, "error.html", h.CommonData(c, gin.H{
			"error": "Cookie export only supported for TikTok",
			"title": "Error",
		}))
		return
	}

	filename := "tiktok_" + source.UserName + ".json"
	filepath := "outputs/tiktok_cookies/" + filename

	c.FileAttachment(filepath, filename)
}

func (h *Handler) HandleImportCookies(c *gin.Context) {
	sourceID, err := uuid.Parse(c.PostForm("source_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid source ID"})
		return
	}

	file, err := c.FormFile("cookie_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	source, err := h.DB.GetSourceById(context.Background(), sourceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Source not found"})
		return
	}

	if source.Network != "TikTok" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cookie import only supported for TikTok"})
		return
	}

	dst := "outputs/tiktok_cookies/tiktok_" + source.UserName + ".json"
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file: " + err.Error()})
		return
	}

	c.Redirect(http.StatusSeeOther, "/sources")
}
