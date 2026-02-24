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
	"github.com/fluffyriot/rpsync/internal/config"
	"github.com/fluffyriot/rpsync/internal/database"
	"github.com/fluffyriot/rpsync/internal/fetcher/common"
	"github.com/google/uuid"
)

type twitchTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type twitchUser struct {
	ID    string `json:"id"`
	Login string `json:"login"`
}

type twitchUsersResponse struct {
	Data []twitchUser `json:"data"`
}

type twitchVideo struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	ViewCount   int    `json:"view_count"`
	Type        string `json:"type"`
	UserLogin   string `json:"user_login"`
}

type twitchVideosResponse struct {
	Data       []twitchVideo `json:"data"`
	Pagination struct {
		Cursor string `json:"cursor"`
	} `json:"pagination"`
}

type twitchClip struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	CreatedAt        string `json:"created_at"`
	ViewCount        int    `json:"view_count"`
	BroadcasterLogin string `json:"broadcaster_login"`
	CreatorName      string `json:"creator_name"`
}

type twitchClipsResponse struct {
	Data       []twitchClip `json:"data"`
	Pagination struct {
		Cursor string `json:"cursor"`
	} `json:"pagination"`
}

type twitchFollowersResponse struct {
	Total int `json:"total"`
}

func twitchGetAppToken(clientID, clientSecret string, c *common.Client) (string, error) {
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", "https://id.twitch.tv/oauth2/token", strings.NewReader(data.Encode()))
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
		return "", fmt.Errorf("Twitch token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp twitchTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode Twitch token response: %w", err)
	}

	return tokenResp.AccessToken, nil
}

func twitchDoRequest(req *http.Request, clientID, appToken string, c *common.Client) ([]byte, int, error) {
	req.Header.Set("Client-Id", clientID)
	req.Header.Set("Authorization", "Bearer "+appToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	return body, resp.StatusCode, nil
}

func twitchGetUserID(username, clientID, appToken string, c *common.Client) (string, error) {
	req, err := http.NewRequest("GET", "https://api.twitch.tv/helix/users?login="+username, nil)
	if err != nil {
		return "", err
	}

	body, status, err := twitchDoRequest(req, clientID, appToken, c)
	if err != nil {
		return "", err
	}

	if status != 200 {
		return "", fmt.Errorf("Twitch users API returned %d: %s", status, string(body))
	}

	var resp twitchUsersResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("failed to decode Twitch users response: %w", err)
	}

	if len(resp.Data) == 0 {
		return "", fmt.Errorf("Twitch user %q not found", username)
	}

	return resp.Data[0].ID, nil
}

func twitchFetchFollowers(userID, clientID, appToken string, c *common.Client) int {
	req, err := http.NewRequest("GET", "https://api.twitch.tv/helix/channels/followers?broadcaster_id="+userID, nil)
	if err != nil {
		log.Printf("Twitch: failed to build followers request: %v", err)
		return 0
	}

	body, status, err := twitchDoRequest(req, clientID, appToken, c)
	if err != nil {
		log.Printf("Twitch: failed to fetch followers: %v", err)
		return 0
	}

	if status != 200 {
		log.Printf("Twitch: followers API returned %d: %s", status, string(body))
		return 0
	}

	var resp twitchFollowersResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		log.Printf("Twitch: failed to decode followers response: %v", err)
		return 0
	}

	return resp.Total
}

func FetchTwitchPosts(dbQueries *database.Queries, encryptionKey []byte, sourceId uuid.UUID, c *common.Client) error {
	ctx := context.Background()

	userSource, err := dbQueries.GetSourceById(ctx, sourceId)
	if err != nil {
		return err
	}
	username := userSource.UserName

	clientSecret, clientID, _, _, err := authhelp.GetSourceToken(ctx, dbQueries, encryptionKey, sourceId)
	if err != nil {
		return fmt.Errorf("Twitch: failed to get credentials: %w", err)
	}

	userAgent := fmt.Sprintf("RPSync/%s (by riotphotos)", config.AppVersion)
	_ = userAgent

	appToken, err := twitchGetAppToken(clientID, clientSecret, c)
	if err != nil {
		return fmt.Errorf("Twitch: failed to get app token: %w", err)
	}

	userID, err := twitchGetUserID(username, clientID, appToken, c)
	if err != nil {
		return err
	}

	exclusionMap, err := common.LoadExclusionMap(dbQueries, sourceId)
	if err != nil {
		return err
	}

	processedPosts := make(map[string]struct{})

	var videoCursor string
	const maxPages = 200
	for page := 0; page < maxPages; page++ {
		time.Sleep(500 * time.Millisecond)

		apiURL := fmt.Sprintf("https://api.twitch.tv/helix/videos?user_id=%s&first=100", userID)
		if videoCursor != "" {
			apiURL += "&after=" + videoCursor
		}

		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return err
		}

		body, status, err := twitchDoRequest(req, clientID, appToken, c)
		if err != nil {
			return err
		}

		if status == 429 {
			log.Printf("Twitch: rate limited on videos, waiting 30s")
			time.Sleep(30 * time.Second)
			continue
		}

		if status != 200 {
			return fmt.Errorf("Twitch videos API returned %d: %s", status, string(body))
		}

		var videosResp twitchVideosResponse
		if err := json.Unmarshal(body, &videosResp); err != nil {
			return fmt.Errorf("failed to decode Twitch videos response: %w", err)
		}

		if len(videosResp.Data) == 0 {
			break
		}

		for _, v := range videosResp.Data {
			if _, exists := processedPosts[v.ID]; exists {
				continue
			}
			processedPosts[v.ID] = struct{}{}

			if exclusionMap[v.ID] {
				continue
			}

			postedAt, err := time.Parse(time.RFC3339, v.CreatedAt)
			if err != nil {
				log.Printf("Twitch: failed to parse time for video %s: %v", v.ID, err)
				postedAt = time.Now()
			}

			postType := "video"
			if v.Type == "archive" {
				postType = "broadcast"
			}

			content := v.Title
			if v.Description != "" {
				content = v.Title + "\n\n" + v.Description
			}

			internalID, err := common.CreateOrUpdatePost(
				ctx,
				dbQueries,
				sourceId,
				v.ID,
				"Twitch",
				postedAt,
				postType,
				username,
				content,
			)
			if err != nil {
				log.Printf("Twitch: failed to save video %s: %v", v.ID, err)
				continue
			}

			_, err = dbQueries.SyncReactions(ctx, database.SyncReactionsParams{
				ID:       uuid.New(),
				SyncedAt: time.Now(),
				PostID:   internalID,
				Likes:    sql.NullInt64{Valid: false},
				Reposts:  sql.NullInt64{Valid: false},
				Views: sql.NullInt64{
					Int64: int64(v.ViewCount),
					Valid: true,
				},
			})
			if err != nil {
				log.Printf("Twitch: failed to sync reactions for video %s: %v", v.ID, err)
			}
		}

		if videosResp.Pagination.Cursor == "" {
			break
		}
		videoCursor = videosResp.Pagination.Cursor
	}

	var clipCursor string
	for page := 0; page < maxPages; page++ {
		time.Sleep(500 * time.Millisecond)

		apiURL := fmt.Sprintf("https://api.twitch.tv/helix/clips?broadcaster_id=%s&first=100", userID)
		if clipCursor != "" {
			apiURL += "&after=" + clipCursor
		}

		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return err
		}

		body, status, err := twitchDoRequest(req, clientID, appToken, c)
		if err != nil {
			return err
		}

		if status == 429 {
			log.Printf("Twitch: rate limited on clips, waiting 30s")
			time.Sleep(30 * time.Second)
			continue
		}

		if status != 200 {
			return fmt.Errorf("Twitch clips API returned %d: %s", status, string(body))
		}

		var clipsResp twitchClipsResponse
		if err := json.Unmarshal(body, &clipsResp); err != nil {
			return fmt.Errorf("failed to decode Twitch clips response: %w", err)
		}

		if len(clipsResp.Data) == 0 {
			break
		}

		for _, clip := range clipsResp.Data {
			if _, exists := processedPosts[clip.ID]; exists {
				continue
			}
			processedPosts[clip.ID] = struct{}{}

			if exclusionMap[clip.ID] {
				continue
			}

			postedAt, err := time.Parse(time.RFC3339, clip.CreatedAt)
			if err != nil {
				log.Printf("Twitch: failed to parse time for clip %s: %v", clip.ID, err)
				postedAt = time.Now()
			}

			internalID, err := common.CreateOrUpdatePost(
				ctx,
				dbQueries,
				sourceId,
				clip.ID,
				"Twitch",
				postedAt,
				"video",
				username,
				clip.Title+"\n\n(TwitchClip by @"+clip.CreatorName+")",
			)
			if err != nil {
				log.Printf("Twitch: failed to save clip %s: %v", clip.ID, err)
				continue
			}

			_, err = dbQueries.SyncReactions(ctx, database.SyncReactionsParams{
				ID:       uuid.New(),
				SyncedAt: time.Now(),
				PostID:   internalID,
				Likes:    sql.NullInt64{Valid: false},
				Reposts:  sql.NullInt64{Valid: false},
				Views: sql.NullInt64{
					Int64: int64(clip.ViewCount),
					Valid: true,
				},
			})
			if err != nil {
				log.Printf("Twitch: failed to sync reactions for clip %s: %v", clip.ID, err)
			}
		}

		if clipsResp.Pagination.Cursor == "" {
			break
		}
		clipCursor = clipsResp.Pagination.Cursor
	}

	if len(processedPosts) == 0 {
		return fmt.Errorf("no content found for Twitch user %q", username)
	}

	avgStats, err := common.CalculateAverageStats(ctx, dbQueries, sourceId)
	if err != nil {
		log.Printf("Twitch: failed to calculate average stats: %v", err)
	} else {
		followers := twitchFetchFollowers(userID, clientID, appToken, c)
		if followers > 0 {
			avgStats.FollowersCount = &followers
		}
		avgStats.FollowingCount = nil

		if err := common.SaveOrUpdateSourceStats(ctx, dbQueries, sourceId, avgStats); err != nil {
			log.Printf("Twitch: failed to save stats: %v", err)
		}
	}

	return nil
}
