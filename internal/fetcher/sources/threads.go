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
	"time"

	"github.com/fluffyriot/rpsync/internal/authhelp"
	"github.com/fluffyriot/rpsync/internal/database"
	"github.com/fluffyriot/rpsync/internal/fetcher/common"
	"github.com/google/uuid"
)

type threadsAPIError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    int    `json:"code"`
		Subcode int    `json:"error_subcode"`
	} `json:"error"`
}

func isThreadsTokenError(body []byte) bool {
	var apiErr threadsAPIError
	if err := json.Unmarshal(body, &apiErr); err != nil {
		return false
	}
	return apiErr.Error.Type == "OAuthException" && apiErr.Error.Code == 190
}

type threadsPost struct {
	ID        string `json:"id"`
	Shortcode string `json:"shortcode"`
	Text      string `json:"text"`
	Timestamp string `json:"timestamp"`
	MediaType string `json:"media_type"`
}

type threadsPostsResponse struct {
	Data   []threadsPost `json:"data"`
	Paging struct {
		Cursors struct {
			After string `json:"after"`
		} `json:"cursors"`
		Next string `json:"next"`
	} `json:"paging"`
}

type threadsInsightValue struct {
	Value int `json:"value"`
}

type threadsInsightMetric struct {
	Name   string                `json:"name"`
	Values []threadsInsightValue `json:"values"`
}

type threadsInsightsResponse struct {
	Data []threadsInsightMetric `json:"data"`
}

func parseThreadsTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02T15:04:05-0700", s)
}

func fetchThreadsFollowers(c *common.Client, accessToken string) (int, error) {
	now := time.Now().UTC()
	since := now.Truncate(24 * time.Hour).Unix()
	until := since + 86400

	url := fmt.Sprintf(
		"https://graph.threads.net/v1.0/me/threads_insights?metric=followers_count&period=day&since=%d&until=%d&access_token=%s",
		since, until, accessToken,
	)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != 200 {
		if isThreadsTokenError(body) {
			return 0, fmt.Errorf("Threads access token is invalid or expired — please update it in Source Settings")
		}
		return 0, fmt.Errorf("threads_insights returned status %d: %s", resp.StatusCode, string(body))
	}

	var insightsResp threadsInsightsResponse
	if err := json.Unmarshal(body, &insightsResp); err != nil {
		return 0, err
	}

	for _, metric := range insightsResp.Data {
		if metric.Name == "followers_count" && len(metric.Values) > 0 {
			return metric.Values[len(metric.Values)-1].Value, nil
		}
	}

	return 0, nil
}

func fetchThreadsInsights(c *common.Client, postID, accessToken string) (likes, reposts, views int, err error) {
	url := fmt.Sprintf(
		"https://graph.threads.net/v1.0/%s/insights?metric=likes,reposts,quotes,views&access_token=%s",
		postID, accessToken,
	)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, 0, 0, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return 0, 0, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, 0, err
	}

	if resp.StatusCode == 403 || resp.StatusCode == 401 {
		return 0, 0, 0, nil
	}
	if resp.StatusCode != 200 {
		return 0, 0, 0, fmt.Errorf("insights returned status %d", resp.StatusCode)
	}

	var insightsResp threadsInsightsResponse
	if err := json.Unmarshal(body, &insightsResp); err != nil {
		return 0, 0, 0, err
	}

	for _, metric := range insightsResp.Data {
		if len(metric.Values) == 0 {
			continue
		}
		switch metric.Name {
		case "likes":
			likes = metric.Values[0].Value
		case "reposts":
			reposts += metric.Values[0].Value
		case "quotes":
			reposts += metric.Values[0].Value
		case "views":
			views = metric.Values[0].Value
		}
	}

	return likes, reposts, views, nil
}

func FetchThreadsPosts(dbQueries *database.Queries, encryptionKey []byte, sourceID uuid.UUID, c *common.Client) error {
	exclusionMap, err := common.LoadExclusionMap(dbQueries, sourceID)
	if err != nil {
		return err
	}

	source, err := dbQueries.GetSourceById(context.Background(), sourceID)
	if err != nil {
		return err
	}
	username := source.UserName

	accessToken, _, _, _, err := authhelp.GetSourceToken(context.Background(), dbQueries, encryptionKey, sourceID)
	if err != nil {
		return fmt.Errorf("failed to get Threads credentials: %w", err)
	}

	followersCount, err := fetchThreadsFollowers(c, accessToken)
	if err != nil {
		log.Printf("Threads: Failed to fetch follower count: %v", err)
	}

	processedIDs := make(map[string]struct{})
	fields := "id,shortcode,text,timestamp,media_type"
	nextURL := fmt.Sprintf(
		"https://graph.threads.net/v1.0/me/threads?fields=%s&limit=100&access_token=%s",
		fields, accessToken,
	)

	const maxPages = 100

	for page := 0; page < maxPages && nextURL != ""; page++ {
		time.Sleep(500 * time.Millisecond)

		req, err := http.NewRequest("GET", nextURL, nil)
		if err != nil {
			return err
		}

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
			log.Printf("Threads: Rate limited, waiting...")
			time.Sleep(5 * time.Second)
			continue
		}

		if resp.StatusCode != 200 {
			if isThreadsTokenError(body) {
				return fmt.Errorf("Threads access token is invalid or expired — please update it in Source Settings")
			}
			return fmt.Errorf("Threads API returned status %d: %s", resp.StatusCode, string(body))
		}

		var postsResp threadsPostsResponse
		if err := json.Unmarshal(body, &postsResp); err != nil {
			return fmt.Errorf("failed to parse Threads response: %w", err)
		}

		if len(postsResp.Data) == 0 {
			break
		}

		for _, post := range postsResp.Data {
			networkID := post.Shortcode
			if networkID == "" {
				networkID = post.ID
			}

			if _, exists := processedIDs[networkID]; exists {
				continue
			}
			processedIDs[networkID] = struct{}{}

			if exclusionMap[networkID] {
				continue
			}

			postedAt, err := parseThreadsTime(post.Timestamp)
			if err != nil {
				log.Printf("Threads: Failed to parse time for post %s (%q): %v", post.ID, post.Timestamp, err)
				postedAt = time.Now()
			}

			if post.MediaType == "REPOST_FACADE" {
				continue
			}

			postType := "post"
			switch post.MediaType {
			case "IMAGE", "CAROUSEL_ALBUM":
				postType = "image"
			case "VIDEO":
				postType = "video"
			}
			author := username
			content := post.Text

			internalID, err := common.CreateOrUpdatePost(
				context.Background(),
				dbQueries,
				sourceID,
				networkID,
				"Threads",
				postedAt,
				postType,
				author,
				content,
			)
			if err != nil {
				log.Printf("Threads: Failed to save post %s: %v", networkID, err)
				continue
			}

			likes, reposts, views, insightErr := fetchThreadsInsights(c, post.ID, accessToken)
			if insightErr != nil {
				log.Printf("Threads: Failed to fetch insights for post %s: %v", post.ID, insightErr)
			}

			_, err = dbQueries.SyncReactions(context.Background(), database.SyncReactionsParams{
				ID:       uuid.New(),
				SyncedAt: time.Now(),
				PostID:   internalID,
				Likes: sql.NullInt64{
					Int64: int64(likes),
					Valid: true,
				},
				Reposts: sql.NullInt64{
					Int64: int64(reposts),
					Valid: reposts > 0,
				},
				Views: sql.NullInt64{
					Int64: int64(views),
					Valid: views > 0,
				},
			})
			if err != nil {
				log.Printf("Threads: Failed to sync reactions for post %s: %v", networkID, err)
			}
		}

		nextURL = postsResp.Paging.Next
	}

	if len(processedIDs) == 0 {
		return fmt.Errorf("no content found for Threads user %s", username)
	}

	avgStats, err := common.CalculateAverageStats(context.Background(), dbQueries, sourceID)
	if err != nil {
		log.Printf("Threads: Failed to calculate average stats: %v", err)
	} else {
		if followersCount > 0 {
			avgStats.FollowersCount = &followersCount
		}
		if err := common.SaveOrUpdateSourceStats(context.Background(), dbQueries, sourceID, avgStats); err != nil {
			log.Printf("Threads: Failed to save stats: %v", err)
		}
	}

	return nil
}
