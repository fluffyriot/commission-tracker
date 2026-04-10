// SPDX-License-Identifier: AGPL-3.0-only
package middleware

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/fluffyriot/rpsync/internal/database"
	"github.com/gin-gonic/gin"
)

func BearerTokenMiddleware(db *database.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid Authorization header"})
			c.Abort()
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "empty bearer token"})
			c.Abort()
			return
		}

		hash := sha256.Sum256([]byte(token))
		tokenHash := hex.EncodeToString(hash[:])

		apiToken, err := db.GetApiTokenByHash(c.Request.Context(), tokenHash)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		c.Set("api_user_id", apiToken.UserID.String())
		c.Set("api_token_id", apiToken.ID.String())

		go func() {
			_ = db.UpdateApiTokenLastUsed(c.Request.Context(), database.UpdateApiTokenLastUsedParams{
				ID:         apiToken.ID,
				LastUsedAt: sql.NullTime{Time: time.Now(), Valid: true},
			})
		}()

		c.Next()
	}
}
