// SPDX-License-Identifier: AGPL-3.0-only
package handlers

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/fluffyriot/rpsync/internal/database"
	"github.com/fluffyriot/rpsync/internal/helpers"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) HandleDownloadTagsCSV(c *gin.Context) {
	if h.Config.DBInitErr != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: h.Config.DBInitErr.Error()})
		return
	}

	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	ctx := c.Request.Context()

	posts, err := h.DB.GetAllPostsWithTheLatestInfoForUser(ctx, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	postTagRows, err := h.DB.GetAllPostTagsForUser(ctx, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	postTagsMap := make(map[uuid.UUID][]string)
	for _, pt := range postTagRows {
		postTagsMap[pt.PostID] = append(postTagsMap[pt.PostID], pt.TagName)
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=tags_template_%s.csv", time.Now().Format("20060102_150405")))

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	_ = writer.Write([]string{
		"post_id",
		"network",
		"post_url",
		"content",
		"tags",
	})

	for _, p := range posts {
		network := ""
		if p.Network.Valid {
			network = p.Network.String
		}

		url := ""
		if network != "" && p.Author != "" {
			url, _ = helpers.ConvPostToURL(network, p.Author, p.NetworkInternalID)
		}

		content := ""
		if p.Content.Valid {
			content = p.Content.String
			if len(content) > 200 {
				content = content[:200] + "..."
			}
		}

		existingTags := ""
		if tags, ok := postTagsMap[p.ID]; ok {
			existingTags = strings.Join(tags, ", ")
		}

		_ = writer.Write([]string{
			p.ID.String(),
			network,
			url,
			content,
			existingTags,
		})
	}
}

func (h *Handler) HandleUploadTagsCSV(c *gin.Context) {
	if h.Config.DBInitErr != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: h.Config.DBInitErr.Error()})
		return
	}

	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "No file uploaded"})
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)

	header, err := reader.Read()
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Failed to read CSV header"})
		return
	}

	postIDIdx := -1
	tagsIdx := -1
	for i, col := range header {
		switch strings.TrimSpace(strings.ToLower(col)) {
		case "post_id":
			postIDIdx = i
		case "tags":
			tagsIdx = i
		}
	}

	if postIDIdx == -1 || tagsIdx == -1 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "CSV must have 'post_id' and 'tags' columns"})
		return
	}

	ctx := c.Request.Context()

	existingTags, err := h.DB.GetTagsForUser(ctx, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	tagNameToID := make(map[string]uuid.UUID)
	for _, t := range existingTags {
		tagNameToID[strings.ToLower(t.Name)] = t.ID
	}

	updatedCount := 0
	skippedCount := 0
	createdTagsCount := 0
	var errors []string

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errors = append(errors, fmt.Sprintf("Error reading row: %v", err))
			continue
		}

		if postIDIdx >= len(record) || tagsIdx >= len(record) {
			skippedCount++
			continue
		}

		postIDStr := strings.TrimSpace(record[postIDIdx])
		tagsStr := strings.TrimSpace(record[tagsIdx])

		if postIDStr == "" || tagsStr == "" {
			skippedCount++
			continue
		}

		postID, err := uuid.Parse(postIDStr)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Invalid post_id '%s'", postIDStr))
			continue
		}

		tagNames := strings.Split(tagsStr, ",")
		var tagIDs []uuid.UUID
		for _, name := range tagNames {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if len(name) > 40 {
				name = name[:40]
			}

			tagID, exists := tagNameToID[strings.ToLower(name)]
			if !exists {
				now := time.Now()
				newTag, err := h.DB.CreateTag(ctx, database.CreateTagParams{
					ID:        uuid.New(),
					CreatedAt: now,
					UpdatedAt: now,
					UserID:    user.ID,
					Name:      name,
				})
				if err != nil {
					log.Printf("Warning: failed to create tag '%s': %v", name, err)
					existingTags2, _ := h.DB.GetTagsForUser(ctx, user.ID)
					for _, t := range existingTags2 {
						if strings.EqualFold(t.Name, name) {
							tagID = t.ID
							tagNameToID[strings.ToLower(name)] = tagID
							exists = true
							break
						}
					}
					if !exists {
						errors = append(errors, fmt.Sprintf("Failed to create tag '%s'", name))
						continue
					}
				} else {
					tagID = newTag.ID
					tagNameToID[strings.ToLower(name)] = tagID
					createdTagsCount++
				}
			}
			tagIDs = append(tagIDs, tagID)
		}

		if len(tagIDs) > 5 {
			tagIDs = tagIDs[:5]
			errors = append(errors, fmt.Sprintf("Post %s: trimmed to 5 tags (max)", postIDStr))
		}

		if err := h.DB.ClearTagsForPost(ctx, postID); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to clear tags for post %s", postIDStr))
			continue
		}

		for _, tagID := range tagIDs {
			_, err := h.DB.AddTagToPost(ctx, database.AddTagToPostParams{
				ID:        uuid.New(),
				CreatedAt: time.Now(),
				PostID:    postID,
				TagID:     tagID,
			})
			if err != nil {
				log.Printf("Warning: failed to add tag to post %s: %v", postIDStr, err)
			}
		}

		updatedCount++
	}

	result := gin.H{
		"message":      fmt.Sprintf("Processed %d posts, skipped %d rows, created %d new tags", updatedCount, skippedCount, createdTagsCount),
		"updated":      updatedCount,
		"skipped":      skippedCount,
		"tags_created": createdTagsCount,
	}
	if len(errors) > 0 {
		if len(errors) > 20 {
			errors = errors[:20]
		}
		result["errors"] = errors
	}

	c.JSON(http.StatusOK, result)
}
