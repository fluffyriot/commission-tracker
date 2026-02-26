// SPDX-License-Identifier: AGPL-3.0-only
package sources

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fluffyriot/rpsync/internal/authhelp"
	"github.com/fluffyriot/rpsync/internal/database"
	"github.com/fluffyriot/rpsync/internal/fetcher/common"
	"github.com/google/uuid"
)

type deviantArtTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Status      string `json:"status"`
}

type deviantArtDeviation struct {
	DeviationID   string `json:"deviationid"`
	Title         string `json:"title"`
	URL           string `json:"url"`
	PublishedTime int64  `json:"published_time"`
	Stats         struct {
		Comments   int `json:"comments"`
		Favourites int `json:"favourites"`
	} `json:"stats"`
}

type deviantArtGalleryResponse struct {
	Results    []deviantArtDeviation `json:"results"`
	HasMore    bool                  `json:"has_more"`
	NextOffset int                   `json:"next_offset"`
}

func getDeviantArtAccessToken(c *common.Client, clientID, clientSecret string) (string, error) {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)

	req, err := http.NewRequest("POST", "https://www.deviantart.com/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("deviantart token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp deviantArtTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse DeviantArt token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("DeviantArt returned empty access token")
	}

	return tokenResp.AccessToken, nil
}

func FetchDeviantArtPosts(dbQueries *database.Queries, encryptionKey []byte, sourceID uuid.UUID, c *common.Client) error {
	exclusionMap, err := common.LoadExclusionMap(dbQueries, sourceID)
	if err != nil {
		return err
	}

	source, err := dbQueries.GetSourceById(context.Background(), sourceID)
	if err != nil {
		return err
	}
	username := source.UserName

	clientSecret, clientID, _, _, err := authhelp.GetSourceToken(context.Background(), dbQueries, encryptionKey, sourceID)
	if err != nil {
		return fmt.Errorf("failed to get DeviantArt credentials: %w", err)
	}

	accessToken, err := getDeviantArtAccessToken(c, clientID, clientSecret)
	if err != nil {
		return fmt.Errorf("failed to obtain DeviantArt access token: %w", err)
	}

	processedIDs := make(map[string]struct{})
	offset := 0
	const limit = 24
	const maxItems = 5000

	for offset < maxItems {
		time.Sleep(500 * time.Millisecond)

		apiURL := fmt.Sprintf(
			"https://www.deviantart.com/api/v1/oauth2/gallery/all?username=%s&limit=%d&offset=%d&mature_content=true",
			url.QueryEscape(username), limit, offset,
		)
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		if resp.StatusCode == 429 {
			log.Printf("DeviantArt: Rate limited, waiting...")
			time.Sleep(5 * time.Second)
			continue
		}

		if resp.StatusCode != 200 {
			return fmt.Errorf("DeviantArt gallery API returned status %d: %s", resp.StatusCode, string(body))
		}

		var galleryResp deviantArtGalleryResponse
		if err := json.Unmarshal(body, &galleryResp); err != nil {
			return fmt.Errorf("failed to parse DeviantArt gallery response: %w", err)
		}

		if len(galleryResp.Results) == 0 {
			break
		}

		for _, deviation := range galleryResp.Results {
			deviationID := deviation.DeviationID

			if _, exists := processedIDs[deviationID]; exists {
				continue
			}
			processedIDs[deviationID] = struct{}{}

			if exclusionMap[deviationID] {
				continue
			}

			postedAt := time.Unix(deviation.PublishedTime, 0)

			internalID, err := common.CreateOrUpdatePost(
				context.Background(),
				dbQueries,
				sourceID,
				deviationID,
				"DeviantArt",
				postedAt,
				"post",
				username,
				deviation.Title,
			)
			if err != nil {
				log.Printf("DeviantArt: Failed to save deviation %s: %v", deviationID, err)
				continue
			}

			_, err = dbQueries.SyncReactions(context.Background(), database.SyncReactionsParams{
				ID:       uuid.New(),
				SyncedAt: time.Now(),
				PostID:   internalID,
				Likes: sql.NullInt64{
					Int64: int64(deviation.Stats.Favourites),
					Valid: true,
				},
				Reposts: sql.NullInt64{Valid: false},
				Views:   sql.NullInt64{Valid: false},
			})
			if err != nil {
				log.Printf("DeviantArt: Failed to sync reactions for %s: %v", deviationID, err)
			}
		}

		if !galleryResp.HasMore {
			break
		}
		offset = galleryResp.NextOffset
	}

	if len(processedIDs) == 0 {
		return fmt.Errorf("no content found for DeviantArt user %s", username)
	}

	avgStats, err := common.CalculateAverageStats(context.Background(), dbQueries, sourceID)
	if err != nil {
		log.Printf("DeviantArt: Failed to calculate average stats: %v", err)
	} else {
		avgStats.FollowersCount = nil
		avgStats.FollowingCount = nil
		if err := common.SaveOrUpdateSourceStats(context.Background(), dbQueries, sourceID, avgStats); err != nil {
			log.Printf("DeviantArt: Failed to save stats: %v", err)
		}
	}

	return nil
}
