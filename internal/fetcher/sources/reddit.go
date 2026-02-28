// SPDX-License-Identifier: AGPL-3.0-only
package sources

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fluffyriot/rpsync/internal/authhelp"
	"github.com/fluffyriot/rpsync/internal/config"
	"github.com/fluffyriot/rpsync/internal/database"
	"github.com/fluffyriot/rpsync/internal/fetcher/common"
	"github.com/google/uuid"
)

var redditHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		TLSNextProto:    make(map[string]func(string, *tls.Conn) http.RoundTripper),
		DisableKeepAlives: true,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	},
}

type redditListing struct {
	Data struct {
		After    string `json:"after"`
		Children []struct {
			Data struct {
				ID         string  `json:"id"`
				Subreddit  string  `json:"subreddit"`
				Title      string  `json:"title"`
				Selftext   string  `json:"selftext"`
				Score      int     `json:"score"`
				CreatedUTC float64 `json:"created_utc"`
				Author     string  `json:"author"`
				IsVideo    bool    `json:"is_video"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

var redditFollowersRe = regexp.MustCompile(`([\d,]+)\s+followers`)

func getRedditDetails(ctx context.Context, dbQueries *database.Queries, encryptionKey []byte, sid uuid.UUID) (username string, subreddits []string, err error) {
	userSource, err := dbQueries.GetSourceById(ctx, sid)
	if err != nil {
		return "", nil, err
	}
	username = userSource.UserName

	_, profileID, _, _, tokenErr := authhelp.GetSourceToken(ctx, dbQueries, encryptionKey, sid)
	if tokenErr == nil && profileID != "" {
		for _, s := range strings.Split(profileID, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				subreddits = append(subreddits, strings.ToLower(s))
			}
		}
	}

	return username, subreddits, nil
}

func fetchRedditFollowers(username, userAgent string) int {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://www.reddit.com/user/%s/", username), nil)
	if err != nil {
		log.Printf("Reddit: failed to build followers request: %v", err)
		return 0
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := redditHTTPClient.Do(req)
	if err != nil {
		log.Printf("Reddit: failed to fetch profile page: %v", err)
		return 0
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Reddit: failed to read profile page body: %v", err)
		return 0
	}

	if resp.StatusCode != 200 {
		log.Printf("Reddit: profile page returned status %d", resp.StatusCode)
		return 0
	}

	idx := strings.Index(string(body), `data-testid="profile-followers-widget"`)
	if idx == -1 {
		log.Printf("Reddit: followers widget not found for user %s", username)
		return 0
	}

	match := redditFollowersRe.FindStringSubmatch(string(body)[idx:])
	if match == nil {
		log.Printf("Reddit: followers count not found in widget for user %s", username)
		return 0
	}

	numStr := strings.ReplaceAll(match[1], ",", "")
	count, err := strconv.Atoi(numStr)
	if err != nil {
		log.Printf("Reddit: failed to parse followers count %q: %v", match[1], err)
		return 0
	}

	return count
}

func handleSubredditChanges(ctx context.Context, dbQueries *database.Queries, sourceID uuid.UUID, newSubreddits []string) {
	if len(newSubreddits) == 0 {
		return
	}

	newSet := make(map[string]bool)
	for _, s := range newSubreddits {
		newSet[strings.ToLower(s)] = true
	}

	posts, err := dbQueries.GetNetworkIdsAndContentBySource(ctx, sourceID)
	if err != nil {
		log.Printf("Reddit: Failed to get existing posts for subreddit pruning: %v", err)
		return
	}

	removedSubreddits := make(map[string]bool)
	for _, post := range posts {
		if !post.Content.Valid {
			continue
		}
		if !strings.HasPrefix(post.Content.String, "r/") {
			continue
		}
		colon := strings.Index(post.Content.String, ":")
		if colon < 3 {
			continue
		}
		subreddit := strings.ToLower(post.Content.String[2:colon])
		if !newSet[subreddit] {
			removedSubreddits[subreddit] = true
		}
	}

	for subreddit := range removedSubreddits {
		pattern := "r/" + subreddit + ":%"
		err := dbQueries.DeletePostsByContentPrefix(ctx, database.DeletePostsByContentPrefixParams{
			SourceID: sourceID,
			Content:  sql.NullString{String: pattern, Valid: true},
		})
		if err != nil {
			log.Printf("Reddit: Failed to delete posts for removed subreddit %s: %v", subreddit, err)
		} else {
			log.Printf("Reddit: Deleted posts for removed subreddit %s", subreddit)
		}
	}
}

func FetchRedditPosts(dbQueries *database.Queries, encryptionKey []byte, sourceId uuid.UUID, _ *common.Client) error {
	ctx := context.Background()

	username, subreddits, err := getRedditDetails(ctx, dbQueries, encryptionKey, sourceId)
	if err != nil {
		return err
	}

	handleSubredditChanges(ctx, dbQueries, sourceId, subreddits)

	userAgent := fmt.Sprintf("rpsync.net:%s (for /u/%s)", config.AppVersion, username)

	exclusionMap, err := common.LoadExclusionMap(dbQueries, sourceId)
	if err != nil {
		return err
	}

	subredditFilter := make(map[string]bool)
	for _, s := range subreddits {
		subredditFilter[s] = true
	}

	processedPosts := make(map[string]struct{})
	var after string
	const maxPages = 500

	for page := 0; page < maxPages; page++ {
		time.Sleep(3 * time.Second)

		apiURL := fmt.Sprintf("https://www.reddit.com/user/%s/submitted.json?limit=100&raw_json=1", username)
		if after != "" {
			apiURL += "&after=" + after
		}

		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return err
		}

		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Accept", "application/json, */*;q=0.9")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")

		resp, err := redditHTTPClient.Do(req)
		if err != nil {
			return err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return err
		}

		if resp.StatusCode == 429 {
			wait := 60 * time.Second
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if secs, err := strconv.ParseFloat(ra, 64); err == nil && secs > 0 {
					wait = time.Duration(secs+5) * time.Second
				}
			}
			log.Printf("Reddit: rate limited, waiting %s", wait)
			time.Sleep(wait)
			continue
		}

		if remaining := resp.Header.Get("X-Ratelimit-Remaining"); remaining != "" {
			if r, err := strconv.ParseFloat(remaining, 64); err == nil && r < 5 {
				reset := resp.Header.Get("X-Ratelimit-Reset")
				wait := 10 * time.Second
				if secs, err := strconv.ParseFloat(reset, 64); err == nil && secs > 0 {
					wait = time.Duration(secs+1) * time.Second
				}
				log.Printf("Reddit: rate limit budget low (%.0f remaining), waiting %s", r, wait)
				time.Sleep(wait)
			}
		}

		if resp.StatusCode != 200 {
			snippet := string(body)
			if len(snippet) > 300 {
				snippet = snippet[:300] + "..."
			}
			return fmt.Errorf("Reddit API returned status %d: %s", resp.StatusCode, snippet)
		}

		var listing redditListing
		if err := json.Unmarshal(body, &listing); err != nil {
			return fmt.Errorf("failed to decode Reddit listing: %w", err)
		}

		if len(listing.Data.Children) == 0 {
			break
		}

		for _, child := range listing.Data.Children {
			post := child.Data
			postID := post.ID

			if _, exists := processedPosts[postID]; exists {
				continue
			}
			processedPosts[postID] = struct{}{}

			if exclusionMap[postID] {
				continue
			}

			if len(subredditFilter) > 0 && !subredditFilter[strings.ToLower(post.Subreddit)] {
				continue
			}

			postedAt := time.Unix(int64(post.CreatedUTC), 0)

			postType := "post"
			if post.IsVideo {
				postType = "video"
			}

			content := fmt.Sprintf("r/%s: %s", post.Subreddit, post.Title)
			if post.Selftext != "" && post.Selftext != "[deleted]" && post.Selftext != "[removed]" {
				content = fmt.Sprintf("r/%s: %s\n\n%s", post.Subreddit, post.Title, post.Selftext)
			}

			internalID, err := common.CreateOrUpdatePost(
				ctx,
				dbQueries,
				sourceId,
				postID,
				"Reddit",
				postedAt,
				postType,
				post.Author,
				content,
			)
			if err != nil {
				log.Printf("Reddit: Failed to save post %s: %v", postID, err)
				continue
			}

			_, err = dbQueries.SyncReactions(ctx, database.SyncReactionsParams{
				ID:       uuid.New(),
				SyncedAt: time.Now(),
				PostID:   internalID,
				Likes: sql.NullInt64{
					Int64: int64(post.Score),
					Valid: true,
				},
				Reposts: sql.NullInt64{Valid: false},
				Views:   sql.NullInt64{Valid: false},
			})
			if err != nil {
				log.Printf("Reddit: Failed to sync reactions for post %s: %v", postID, err)
			}
		}

		if listing.Data.After == "" {
			break
		}

		after = listing.Data.After
	}

	if len(processedPosts) == 0 {
		return errors.New("no posts found for Reddit user")
	}

	avgStats, err := common.CalculateAverageStats(ctx, dbQueries, sourceId)
	if err != nil {
		log.Printf("Reddit: Failed to calculate average stats: %v", err)
	} else {
		followers := fetchRedditFollowers(username, userAgent)
		if followers > 0 {
			avgStats.FollowersCount = &followers
		}

		if err := common.SaveOrUpdateSourceStats(ctx, dbQueries, sourceId, avgStats); err != nil {
			log.Printf("Reddit: Failed to save stats: %v", err)
		}
	}

	return nil
}
