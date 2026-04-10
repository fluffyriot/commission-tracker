// SPDX-License-Identifier: AGPL-3.0-only
package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/fluffyriot/rpsync/internal/backup"
	"github.com/fluffyriot/rpsync/internal/database"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) BackupExportHandler(c *gin.Context) {
	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	ctx := c.Request.Context()

	exportRecord, err := h.DB.CreateExport(ctx, database.CreateExportParams{
		ID:           uuid.New(),
		CreatedAt:    time.Now(),
		CompletedAt:  time.Time{},
		ExportStatus: "Processing",
		StatusMessage: sql.NullString{
			String: "Creating full backup...",
			Valid:  true,
		},
		UserID:       user.ID,
		DownloadUrl:  sql.NullString{},
		ExportMethod: "backup",
		TargetID:     uuid.NullUUID{},
	})
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", h.CommonData(c, gin.H{
			"error": "Failed to create export record",
			"title": "Error",
		}))
		return
	}

	go func(userID uuid.UUID, exportID uuid.UUID) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in backup export: %v", r)
			}
		}()

		bgCtx := context.Background()
		zipPath, err := backup.ExportUserData(bgCtx, h.DB, userID, exportID)
		if err != nil {
			log.Printf("backup export failed: %v", err)
			h.DB.ChangeExportStatusById(bgCtx, database.ChangeExportStatusByIdParams{
				ID:            exportID,
				ExportStatus:  "Failed",
				StatusMessage: sql.NullString{String: err.Error(), Valid: true},
				DownloadUrl:   sql.NullString{},
				CompletedAt:   time.Now(),
			})
			return
		}

		filename := filepath.Base(zipPath)
		downloadUrl := fmt.Sprintf("/outputs/%s", filename)
		h.DB.ChangeExportStatusById(bgCtx, database.ChangeExportStatusByIdParams{
			ID:            exportID,
			ExportStatus:  "Completed",
			StatusMessage: sql.NullString{String: "Full backup created successfully", Valid: true},
			DownloadUrl:   sql.NullString{String: downloadUrl, Valid: true},
			CompletedAt:   time.Now(),
		})
	}(user.ID, exportRecord.ID)

	c.Redirect(http.StatusSeeOther, "/exports")
}

func (h *Handler) BackupImportHandler(c *gin.Context) {
	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	mode := backup.ImportMode(c.PostForm("mode"))
	if mode != backup.ImportModeReplace && mode != backup.ImportModeNew {
		c.HTML(http.StatusBadRequest, "error.html", h.CommonData(c, gin.H{
			"error": "Invalid import mode. Must be 'replace' or 'new'.",
			"title": "Error",
		}))
		return
	}

	file, _, err := c.Request.FormFile("backup_file")
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", h.CommonData(c, gin.H{
			"error": "Please select a backup file to upload.",
			"title": "Error",
		}))
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", h.CommonData(c, gin.H{
			"error": "Failed to read uploaded file.",
			"title": "Error",
		}))
		return
	}

	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", h.CommonData(c, gin.H{
			"error": "Invalid ZIP file. Please upload a valid RPSync backup.",
			"title": "Error",
		}))
		return
	}

	ctx := c.Request.Context()
	result, err := backup.ImportUserData(ctx, h.DB, h.DBConn, zipReader, mode, user.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", h.CommonData(c, gin.H{
			"error": fmt.Sprintf("Import failed: %v", err),
			"title": "Error",
		}))
		return
	}

	c.SetCookie("backup_import_success", "true", 3600, "/", "", false, true)
	if result.GeneratedUsername != "" {
		c.SetCookie("generated_username", result.GeneratedUsername, 3600, "/", "", false, true)
	}
	c.Redirect(http.StatusSeeOther, "/settings/sync")
}

func (h *Handler) BackupRestoreHandler(c *gin.Context) {
	ctx := c.Request.Context()

	users, err := h.DB.GetAllUsers(ctx)
	if err != nil || len(users) > 0 {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	file, _, err := c.Request.FormFile("backup_file")
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", h.CommonData(c, gin.H{
			"error":        "Please select a backup file to upload.",
			"title":        "Error",
			"is_auth_page": true,
		}))
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", h.CommonData(c, gin.H{
			"error":        "Failed to read uploaded file.",
			"title":        "Error",
			"is_auth_page": true,
		}))
		return
	}

	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", h.CommonData(c, gin.H{
			"error":        "Invalid ZIP file. Please upload a valid RPSync backup.",
			"title":        "Error",
			"is_auth_page": true,
		}))
		return
	}

	result, err := backup.ImportUserData(ctx, h.DB, h.DBConn, zipReader, backup.ImportModeNew, uuid.Nil)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", h.CommonData(c, gin.H{
			"error":        fmt.Sprintf("Restore failed: %v", err),
			"title":        "Error",
			"is_auth_page": true,
		}))
		return
	}

	message := fmt.Sprintf("Backup restored successfully! Imported %d sources, %d posts, %d reactions. Please log in to set your password.", result.Sources, result.Posts, result.Reactions)
	if result.GeneratedUsername != "" {
		message = fmt.Sprintf("Backup restored with username: %s (original username was taken). %d sources, %d posts, %d reactions imported. Please log in to set your password.", result.GeneratedUsername, result.Sources, result.Posts, result.Reactions)
	}

	c.HTML(http.StatusOK, "user-setup.html", h.CommonData(c, gin.H{
		"title":        "Restore Complete",
		"is_auth_page": true,
		"Message1":     message,
		"restore_done": true,
	}))
}
