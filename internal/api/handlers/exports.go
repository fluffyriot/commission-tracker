package handlers

import (
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/fluffyriot/rpsync/internal/database"
	"github.com/fluffyriot/rpsync/internal/exports"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) ExportsHandler(c *gin.Context) {

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

	exports, err := h.DB.GetLast20ExportsByUserId(ctx, user.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", h.CommonData(c, gin.H{
			"error": err.Error(),
			"title": "Error",
		}))
		return
	}
	c.HTML(http.StatusOK, "exports.html", h.CommonData(c, gin.H{
		"username": user.Username,
		"user_id":  user.ID,
		"exports":  exports,
		"title":    "Exports",
	}))
}

func (h *Handler) ExportDeleteAllHandler(c *gin.Context) {
	userId, err := uuid.Parse(c.PostForm("user_id"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", h.CommonData(c, gin.H{
			"error": err.Error(),
			"title": "Error",
		}))
		return
	}

	go func(uid uuid.UUID) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in background sync: %v", r)
			}
		}()
		exports.DeleteAllExports(uid, h.DB)
	}(userId)

	c.Redirect(http.StatusSeeOther, "/")
}

func (h *Handler) DownloadExportHandler(c *gin.Context) {
	ctx := c.Request.Context()
	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	requestedFilename := c.Param("filepath")[1:]
	requestedFilename = filepath.Clean(requestedFilename)

	userExports, err := h.DB.GetAllExportsByUserId(ctx, user.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", h.CommonData(c, gin.H{
			"error": "Internal server error fetching exports",
			"title": "Error",
		}))
		return
	}

	var matchedExport *database.Export
	for _, exp := range userExports {
		if exp.DownloadUrl.Valid {

			storedPath := exp.DownloadUrl.String
			storedFilename := filepath.Base(storedPath)

			if storedFilename == requestedFilename {
				matchedExport = &exp
				break
			}
		}
	}

	if matchedExport == nil {
		c.HTML(http.StatusForbidden, "error.html", h.CommonData(c, gin.H{
			"error": "Access denied",
			"title": "Error",
		}))
		return
	}

	baseDir, err := filepath.Abs("./outputs")
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", h.CommonData(c, gin.H{
			"error": "Internal server error resolving base path",
			"title": "Error",
		}))
		return
	}

	fullPath := filepath.Join(baseDir, requestedFilename)

	if !strings.HasPrefix(fullPath, baseDir) {
		c.HTML(http.StatusForbidden, "error.html", h.CommonData(c, gin.H{
			"error": "Access denied",
			"title": "Error",
		}))
		return
	}

	c.FileAttachment(fullPath, requestedFilename)
}
