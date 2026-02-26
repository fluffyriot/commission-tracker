// SPDX-License-Identifier: AGPL-3.0-only
package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/fluffyriot/rpsync/internal/authhelp"
	"github.com/fluffyriot/rpsync/internal/database"
	"github.com/fluffyriot/rpsync/internal/fetcher/common"
	"github.com/google/uuid"
)

type weasylSubmission struct {
	SubmitID  int    `json:"submitid"`
	Title     string `json:"title"`
	PostedAt  string `json:"posted_at"`
	Link      string `json:"link"`
	Rating    string `json:"rating"`
	SubType   string `json:"subtype"`
}

type weasylSubmissionsResponse struct {
	Submissions []weasylSubmission `json:"submissions"`
	NextID      int                `json:"nextid"`
}

type weasylUserView struct {
	UserInfo struct {
		Watchers int `json:"watchers"`
		Watching int `json:"watching"`
	} `json:"user_info"`
}

func fetchWeasylProfile(c *common.Client, username, apiKey string) (*weasylUserView, error) {
	req, err := http.NewRequest("GET", "https://www.weasyl.com/api/users/"+username+"/view", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Weasyl-API-Key", apiKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("weasyl profile request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var profile weasylUserView
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, err
	}

	return &profile, nil
}

func FetchWeasylPosts(dbQueries *database.Queries, encryptionKey []byte, sourceID uuid.UUID, c *common.Client) error {
	exclusionMap, err := common.LoadExclusionMap(dbQueries, sourceID)
	if err != nil {
		return err
	}

	source, err := dbQueries.GetSourceById(context.Background(), sourceID)
	if err != nil {
		return err
	}
	username := source.UserName

	apiKey, _, _, _, err := authhelp.GetSourceToken(context.Background(), dbQueries, encryptionKey, sourceID)
	if err != nil {
		return fmt.Errorf("failed to get Weasyl credentials: %w", err)
	}

	profile, err := fetchWeasylProfile(c, username, apiKey)
	if err != nil {
		log.Printf("Weasyl: Failed to fetch profile stats: %v", err)
	}

	processedIDs := make(map[string]struct{})
	var nextID int
	const maxPages = 500

	for page := 0; page < maxPages; page++ {
		time.Sleep(500 * time.Millisecond)

		apiURL := fmt.Sprintf("https://www.weasyl.com/api/submissions/user/%s?count=100", username)
		if nextID > 0 {
			apiURL += fmt.Sprintf("&nextid=%d", nextID)
		}

		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("X-Weasyl-API-Key", apiKey)

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
			log.Printf("Weasyl: Rate limited, waiting...")
			time.Sleep(5 * time.Second)
			continue
		}

		if resp.StatusCode != 200 {
			return fmt.Errorf("Weasyl API returned status %d: %s", resp.StatusCode, string(body))
		}

		var submResp weasylSubmissionsResponse
		if err := json.Unmarshal(body, &submResp); err != nil {
			return fmt.Errorf("failed to parse Weasyl response: %w", err)
		}

		if len(submResp.Submissions) == 0 {
			break
		}

		for _, sub := range submResp.Submissions {
			submitID := strconv.Itoa(sub.SubmitID)

			if _, exists := processedIDs[submitID]; exists {
				continue
			}
			processedIDs[submitID] = struct{}{}

			if exclusionMap[submitID] {
				continue
			}

			postedAt, err := time.Parse(time.RFC3339, sub.PostedAt)
			if err != nil {
				log.Printf("Weasyl: Failed to parse time for submission %d: %v", sub.SubmitID, err)
				postedAt = time.Now()
			}

			_, err = common.CreateOrUpdatePost(
				context.Background(),
				dbQueries,
				sourceID,
				submitID,
				"Weasyl",
				postedAt,
				"post",
				username,
				sub.Title,
			)
			if err != nil {
				log.Printf("Weasyl: Failed to save submission %s: %v", submitID, err)
				continue
			}
		}

		if submResp.NextID == 0 {
			break
		}
		nextID = submResp.NextID
	}

	if len(processedIDs) == 0 {
		return fmt.Errorf("no content found for Weasyl user %s", username)
	}

	avgStats, err := common.CalculateAverageStats(context.Background(), dbQueries, sourceID)
	if err != nil {
		log.Printf("Weasyl: Failed to calculate average stats: %v", err)
	} else {
		if profile != nil {
			watchers := profile.UserInfo.Watchers
			watching := profile.UserInfo.Watching
			avgStats.FollowersCount = &watchers
			avgStats.FollowingCount = &watching
		}
		if err := common.SaveOrUpdateSourceStats(context.Background(), dbQueries, sourceID, avgStats); err != nil {
			log.Printf("Weasyl: Failed to save stats: %v", err)
		}
	}

	return nil
}
