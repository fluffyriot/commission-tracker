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
	"strconv"
	"strings"
	"time"

	"github.com/fluffyriot/rpsync/internal/authhelp"
	"github.com/fluffyriot/rpsync/internal/database"
	"github.com/fluffyriot/rpsync/internal/fetcher/common"
	"github.com/google/uuid"
)

type flexInt64 int64

func (f *flexInt64) UnmarshalJSON(data []byte) error {
	var n int64
	if err := json.Unmarshal(data, &n); err == nil {
		*f = flexInt64(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("flexInt64: cannot parse %q as int64: %w", s, err)
	}
	*f = flexInt64(n)
	return nil
}

type deviantArtTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Status      string `json:"status"`
}

type deviantArtDeviation struct {
	DeviationID   string    `json:"deviationid"`
	Title         string    `json:"title"`
	URL           string    `json:"url"`
	PublishedTime flexInt64 `json:"published_time"`
	Stats         struct {
		Favourites int `json:"favourites"`
	} `json:"stats"`
}

type deviantArtGalleryResponse struct {
	Results    []deviantArtDeviation `json:"results"`
	HasMore    bool                  `json:"has_more"`
	NextOffset int                   `json:"next_offset"`
}

type deviantArtMetadataEntry struct {
	DeviationID string `json:"deviationid"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Tags        []struct {
		TagName string `json:"tag_name"`
	} `json:"tags"`
	Stats struct {
		Views      int `json:"views"`
		Favourites int `json:"favourites"`
	} `json:"stats"`
}

type deviantArtMetadataResponse struct {
	Metadata []deviantArtMetadataEntry `json:"metadata"`
}

type deviantArtWatchersResponse struct {
	HasMore    bool              `json:"has_more"`
	NextOffset int               `json:"next_offset"`
	Results    []json.RawMessage `json:"results"`
}

func fetchWatcherCount(c *common.Client, accessToken, username string) (int, error) {
	total := 0
	offset := 0
	for {
		params := url.Values{}
		params.Set("limit", "50")
		params.Set("offset", strconv.Itoa(offset))
		params.Set("mature_content", "true")

		apiURL := fmt.Sprintf(
			"https://www.deviantart.com/api/v1/oauth2/user/watchers/%s?%s",
			url.PathEscape(username), params.Encode(),
		)
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return total, err
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return total, err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return total, err
		}

		if resp.StatusCode == 429 {
			log.Printf("DeviantArt: Rate limited on watchers, waiting...")
			time.Sleep(5 * time.Second)
			continue
		}
		if resp.StatusCode != 200 {
			return total, fmt.Errorf("DeviantArt watchers API returned status %d: %s", resp.StatusCode, string(body))
		}

		var watchResp deviantArtWatchersResponse
		if err := json.Unmarshal(body, &watchResp); err != nil {
			return total, fmt.Errorf("failed to parse DeviantArt watchers response: %w", err)
		}

		total += len(watchResp.Results)

		if !watchResp.HasMore || len(watchResp.Results) == 0 {
			break
		}
		offset = watchResp.NextOffset
		time.Sleep(300 * time.Millisecond)
	}
	return total, nil
}

func deviationSlugFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Path == "" {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func fetchDeviationMetadata(c *common.Client, accessToken string, ids []string) (map[string]deviantArtMetadataEntry, error) {
	params := url.Values{}
	for _, id := range ids {
		params.Add("deviationids[]", id)
	}
	params.Set("ext_submission", "false")
	params.Set("ext_camera", "false")
	params.Set("ext_stats", "true")
	params.Set("ext_collection", "false")
	params.Set("ext_gallery", "false")
	params.Set("with_session", "false")
	params.Set("mature_content", "true")

	req, err := http.NewRequest("GET", "https://www.deviantart.com/api/v1/oauth2/deviation/metadata?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("DeviantArt metadata API returned status %d: %s", resp.StatusCode, string(body))
	}

	var metaResp deviantArtMetadataResponse
	if err := json.Unmarshal(body, &metaResp); err != nil {
		return nil, fmt.Errorf("failed to parse DeviantArt metadata response: %w", err)
	}

	result := make(map[string]deviantArtMetadataEntry, len(metaResp.Metadata))
	for _, entry := range metaResp.Metadata {
		result[strings.ToUpper(entry.DeviationID)] = entry
	}
	return result, nil
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

		uuids := make([]string, 0, len(galleryResp.Results))
		for _, d := range galleryResp.Results {
			uuids = append(uuids, d.DeviationID)
		}
		metadata, err := fetchDeviationMetadata(c, accessToken, uuids)
		if err != nil {
			log.Printf("DeviantArt: failed to fetch metadata for page at offset %d: %v", offset, err)
			metadata = make(map[string]deviantArtMetadataEntry)
		}

		for _, deviation := range galleryResp.Results {
			slug := deviationSlugFromURL(deviation.URL)
			if slug == "" {
				slug = deviation.DeviationID
			}

			if _, exists := processedIDs[slug]; exists {
				continue
			}
			processedIDs[slug] = struct{}{}

			if exclusionMap[slug] {
				continue
			}

			meta, hasMeta := metadata[strings.ToUpper(deviation.DeviationID)]

			var sb strings.Builder
			sb.WriteString(deviation.Title)
			if hasMeta && meta.Description != "" {
				desc := common.StripHTMLToText(meta.Description)
				if desc != "" {
					sb.WriteString("\n\n")
					sb.WriteString(desc)
				}
			}
			if hasMeta {
				sb.WriteString("\n\n")
				for _, tag := range meta.Tags {
					sb.WriteString(" #")
					sb.WriteString(strings.ReplaceAll(tag.TagName, " ", "_"))
				}
			}

			postedAt := time.Unix(int64(deviation.PublishedTime), 0)

			favourites := deviation.Stats.Favourites
			views := 0
			if hasMeta {
				favourites = meta.Stats.Favourites
				views = meta.Stats.Views
			}

			internalID, err := common.CreateOrUpdatePost(
				context.Background(),
				dbQueries,
				sourceID,
				slug,
				"DeviantArt",
				postedAt,
				"post",
				username,
				sb.String(),
			)
			if err != nil {
				log.Printf("DeviantArt: Failed to save deviation %s: %v", slug, err)
				continue
			}

			_, err = dbQueries.SyncReactions(context.Background(), database.SyncReactionsParams{
				ID:       uuid.New(),
				SyncedAt: time.Now(),
				PostID:   internalID,
				Likes: sql.NullInt64{
					Int64: int64(favourites),
					Valid: true,
				},
				Reposts: sql.NullInt64{Valid: false},
				Views: sql.NullInt64{
					Int64: int64(views),
					Valid: hasMeta,
				},
			})
			if err != nil {
				log.Printf("DeviantArt: Failed to sync reactions for %s: %v", slug, err)
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

	watcherCount, err := fetchWatcherCount(c, accessToken, username)
	if err != nil {
		log.Printf("DeviantArt: Failed to fetch watcher count: %v", err)
	}

	avgStats, err := common.CalculateAverageStats(context.Background(), dbQueries, sourceID)
	if err != nil {
		log.Printf("DeviantArt: Failed to calculate average stats: %v", err)
	} else {
		avgStats.FollowingCount = nil
		avgStats.FollowersCount = &watcherCount
		if err := common.SaveOrUpdateSourceStats(context.Background(), dbQueries, sourceID, avgStats); err != nil {
			log.Printf("DeviantArt: Failed to save stats: %v", err)
		}
	}

	return nil
}
