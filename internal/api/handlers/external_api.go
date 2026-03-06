// SPDX-License-Identifier: AGPL-3.0-only
package handlers

import (
	"net/http"
	"time"

	"github.com/fluffyriot/rpsync/internal/database"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) getAPIUserID(c *gin.Context) (uuid.UUID, bool) {
	userIDStr, exists := c.Get("api_user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return uuid.UUID{}, false
	}
	uid, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return uuid.UUID{}, false
	}
	return uid, true
}

func (h *Handler) ExternalAPIFollowersHandler(c *gin.Context) {
	userID, ok := h.getAPIUserID(c)
	if !ok {
		return
	}

	ctx := c.Request.Context()

	current, err := h.DB.GetCurrentTotalFollowers(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get follower count"})
		return
	}

	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	previous, err := h.DB.GetTotalFollowersAtDate(ctx, database.GetTotalFollowersAtDateParams{
		UserID: userID,
		Date:   sevenDaysAgo,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get historical follower count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"followers": current,
		"delta_7d":  current - previous,
	})
}

func (h *Handler) ExternalAPIStatsHandler(c *gin.Context) {
	userID, ok := h.getAPIUserID(c)
	if !ok {
		return
	}

	ctx := c.Request.Context()

	currentStats, err := h.DB.GetCurrentTotalStats(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get current stats"})
		return
	}

	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	previousStats, err := h.DB.GetTotalStatsAtDate(ctx, database.GetTotalStatsAtDateParams{
		UserID:   userID,
		SyncedAt: sevenDaysAgo,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get historical stats"})
		return
	}

	currentEngagement := currentStats.TotalLikes + currentStats.TotalReposts
	previousEngagement := previousStats.TotalLikes + previousStats.TotalReposts

	c.JSON(http.StatusOK, gin.H{
		"likes":      gin.H{"current": currentStats.TotalLikes, "delta_7d": currentStats.TotalLikes - previousStats.TotalLikes},
		"reposts":    gin.H{"current": currentStats.TotalReposts, "delta_7d": currentStats.TotalReposts - previousStats.TotalReposts},
		"engagement": gin.H{"current": currentEngagement, "delta_7d": currentEngagement - previousEngagement},
		"views":      gin.H{"current": currentStats.TotalViews, "delta_7d": currentStats.TotalViews - previousStats.TotalViews},
	})
}

func (h *Handler) ExternalAPIStatusHandler(c *gin.Context) {
	userID, ok := h.getAPIUserID(c)
	if !ok {
		return
	}

	ctx := c.Request.Context()

	sourceCounts, err := h.DB.GetSourceStatusCounts(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get source status"})
		return
	}

	targetCounts, err := h.DB.GetTargetStatusCounts(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get target status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sources": gin.H{"healthy": sourceCounts.HealthyCount, "enabled": sourceCounts.EnabledCount},
		"targets": gin.H{"healthy": targetCounts.HealthyCount, "enabled": targetCounts.EnabledCount},
	})
}
