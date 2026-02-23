// SPDX-License-Identifier: AGPL-3.0-only
package handlers

import (
	"strings"

	"github.com/fluffyriot/rpsync/internal/authhelp"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UpdateChannelsRequest struct {
	Channels []string `json:"channels" binding:"required"`
}

type GetSourceChannelsResponse struct {
	Network  string   `json:"network"`
	Channels []string `json:"channels"`
}

func (h *Handler) GetSourceChannelsHandler(c *gin.Context) {
	sourceIDStr := c.Param("source_id")
	if sourceIDStr == "" {
		c.JSON(400, gin.H{"error": "source_id is required"})
		return
	}

	sourceID, err := uuid.Parse(sourceIDStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid source ID"})
		return
	}

	source, err := h.DB.GetSourceById(c.Request.Context(), sourceID)
	if err != nil {
		c.JSON(404, gin.H{"error": "Source not found"})
		return
	}

	_, profileID, _, _, err := authhelp.GetSourceToken(
		c.Request.Context(),
		h.DB,
		h.Config.TokenEncryptionKey,
		sourceID,
	)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get source configuration"})
		return
	}

	switch source.Network {
	case "Discord":
		parts := strings.SplitN(profileID, ":::", 2)
		if len(parts) != 2 {
			c.JSON(200, GetSourceChannelsResponse{Network: "Discord", Channels: []string{}})
			return
		}
		channelIDs := []string{}
		for _, id := range strings.Split(parts[1], ",") {
			id = strings.TrimSpace(id)
			if id != "" {
				channelIDs = append(channelIDs, id)
			}
		}
		c.JSON(200, GetSourceChannelsResponse{Network: "Discord", Channels: channelIDs})

	case "Reddit":
		subreddits := []string{}
		for _, s := range strings.Split(profileID, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				subreddits = append(subreddits, s)
			}
		}
		c.JSON(200, GetSourceChannelsResponse{Network: "Reddit", Channels: subreddits})

	default:
		c.JSON(400, gin.H{"error": "Channel management not supported for this network"})
	}
}

func (h *Handler) UpdateSourceChannelsHandler(c *gin.Context) {
	sourceIDStr := c.Param("source_id")
	if sourceIDStr == "" {
		c.JSON(400, gin.H{"error": "source_id is required"})
		return
	}

	sourceID, err := uuid.Parse(sourceIDStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid source ID"})
		return
	}

	source, err := h.DB.GetSourceById(c.Request.Context(), sourceID)
	if err != nil {
		c.JSON(404, gin.H{"error": "Source not found"})
		return
	}

	var req UpdateChannelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	_, profileID, _, _, err := authhelp.GetSourceToken(
		c.Request.Context(),
		h.DB,
		h.Config.TokenEncryptionKey,
		sourceID,
	)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get source configuration"})
		return
	}

	switch source.Network {
	case "Discord":
		if len(req.Channels) == 0 {
			c.JSON(400, gin.H{"error": "At least one channel ID is required"})
			return
		}

		parts := strings.SplitN(profileID, ":::", 2)
		if len(parts) != 2 {
			c.JSON(500, gin.H{"error": "Invalid source configuration"})
			return
		}
		serverID := parts[0]

		cleaned := make([]string, 0, len(req.Channels))
		for _, ch := range req.Channels {
			ch = strings.TrimSpace(ch)
			if ch != "" {
				cleaned = append(cleaned, ch)
			}
		}

		err = authhelp.UpdateSourceProfile(
			c.Request.Context(),
			h.DB,
			h.Config.TokenEncryptionKey,
			sourceID,
			serverID+":::"+strings.Join(cleaned, ","),
		)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to update channels: " + err.Error()})
			return
		}

		c.JSON(200, gin.H{"success": true, "message": "Channels updated successfully", "channels": cleaned})

	case "Reddit":
		cleaned := make([]string, 0, len(req.Channels))
		for _, s := range req.Channels {
			s = strings.TrimSpace(s)
			if s != "" {
				cleaned = append(cleaned, s)
			}
		}

		err = authhelp.UpdateSourceProfile(
			c.Request.Context(),
			h.DB,
			h.Config.TokenEncryptionKey,
			sourceID,
			strings.Join(cleaned, ","),
		)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to update subreddits: " + err.Error()})
			return
		}

		c.JSON(200, gin.H{"success": true, "message": "Subreddits updated successfully", "channels": cleaned})

	default:
		c.JSON(400, gin.H{"error": "Channel management not supported for this network"})
	}
}
