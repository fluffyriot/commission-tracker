// SPDX-License-Identifier: AGPL-3.0-only
package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/fluffyriot/rpsync/internal/database"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func generateToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return "rps_" + hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (h *Handler) HandleCreateApiToken(c *gin.Context) {
	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	token, err := generateToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	_, err = h.DB.CreateApiToken(c.Request.Context(), database.CreateApiTokenParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UserID:    user.ID,
		TokenHash: hashToken(token),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (h *Handler) HandleGetApiTokens(c *gin.Context) {
	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	tokens, err := h.DB.GetApiTokensByUser(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get tokens"})
		return
	}

	type tokenResponse struct {
		ID         string  `json:"id"`
		CreatedAt  string  `json:"created_at"`
		LastUsedAt *string `json:"last_used_at"`
	}

	result := make([]tokenResponse, len(tokens))
	for i, t := range tokens {
		resp := tokenResponse{
			ID:        t.ID.String(),
			CreatedAt: t.CreatedAt.Format(time.RFC3339),
		}
		if t.LastUsedAt.Valid {
			s := t.LastUsedAt.Time.Format(time.RFC3339)
			resp.LastUsedAt = &s
		}
		result[i] = resp
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) HandleDeleteApiToken(c *gin.Context) {
	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	tokenID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token id"})
		return
	}

	err = h.DB.DeleteApiToken(c.Request.Context(), database.DeleteApiTokenParams{
		ID:     tokenID,
		UserID: user.ID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "token deleted"})
}
