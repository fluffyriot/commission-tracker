// SPDX-License-Identifier: AGPL-3.0-only
package sources

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/fluffyriot/rpsync/internal/database"
	"github.com/fluffyriot/rpsync/internal/fetcher/common"
	"github.com/google/uuid"
)

type furAffinityProfile struct {
	FollowersCount int
	FollowingCount int
}

func fetchFurAffinityProfile(c *common.Client, username string) (*furAffinityProfile, error) {
	url := fmt.Sprintf("https://www.furaffinity.net/user/%s/", username)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch profile: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	profile := &furAffinityProfile{}

	doc.Find(".section-header .floatright h3 a").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		
		if strings.Contains(text, "Watched by") {
			parts := strings.Fields(text)
			if len(parts) > 0 {
				lastPart := parts[len(parts)-1]
				numberStr := strings.TrimSuffix(lastPart, ")")
				count, _ := strconv.Atoi(numberStr)
				profile.FollowersCount = count
			}
		} else if strings.Contains(text, "Watching") {
			parts := strings.Fields(text)
			if len(parts) > 0 {
				lastPart := parts[len(parts)-1]
				numberStr := strings.TrimSuffix(lastPart, ")")
				count, _ := strconv.Atoi(numberStr)
				profile.FollowingCount = count
			}
		}
	})

	return profile, nil
}

func FetchFurAffinityPosts(dbQueries *database.Queries, c *common.Client, uid uuid.UUID, sourceId uuid.UUID) error {
	exclusionMap, err := common.LoadExclusionMap(dbQueries, sourceId)
	if err != nil {
		return err
	}

	userSource, err := dbQueries.GetSourceById(
		context.Background(),
		sourceId,
	)
	if err != nil {
		return err
	}
	username := userSource.UserName

	profile, err := fetchFurAffinityProfile(c, username)
	if err != nil {
		log.Printf("FurAffinity: Failed to fetch profile stats: %v", err)
	}

	defer func() {
		stats, err := common.CalculateAverageStats(context.Background(), dbQueries, sourceId)
		if err != nil {
			log.Printf("FurAffinity: Failed to calculate stats for source %s: %v", sourceId, err)
		} else {
			if profile != nil {
				stats.FollowersCount = &profile.FollowersCount
				stats.FollowingCount = &profile.FollowingCount
			}
			if err := common.SaveOrUpdateSourceStats(context.Background(), dbQueries, sourceId, stats); err != nil {
				log.Printf("FurAffinity: Failed to save stats for source %s: %v", sourceId, err)
			}
		}
	}()

	processedLinks := make(map[string]struct{})
	page := 1
	const maxPages = 500

	for page <= maxPages {
		galleryUrl := fmt.Sprintf("https://www.furaffinity.net/gallery/%s/%d/?", username, page)
		req, err := http.NewRequest("GET", galleryUrl, nil)
		if err != nil {
			return err
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return fmt.Errorf("failed to fetch gallery page %d: %d", page, resp.StatusCode)
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			return err
		}

		submissionLinks := doc.Find("figure figcaption a")
		if submissionLinks.Length() == 0 {
			break
		}

		foundNew := false
		submissionLinks.Each(func(i int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if !exists {
				return
			}

			parts := strings.Split(href, "/")
			if len(parts) < 3 {
				return
			}
			submissionId := parts[2]

			if _, exists := processedLinks[submissionId]; exists {
				return
			}
			processedLinks[submissionId] = struct{}{}
			
			if exclusionMap[submissionId] {
				return
			}

			err := processSubmission(dbQueries, c, sourceId, submissionId, username)
			if err != nil {
				log.Printf("FurAffinity: Failed to process submission %s: %v", submissionId, err)
			} else {
				foundNew = true
			}
		})

		if !foundNew {
		}
		
		page++
	}
	
	if len(processedLinks) == 0 {
		return fmt.Errorf("no content found")
	}

	return nil
}

func processSubmission(dbQueries *database.Queries, c *common.Client, sourceId uuid.UUID, submissionId string, username string) error {
	url := fmt.Sprintf("https://www.furaffinity.net/view/%s/", submissionId)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to fetch submission %s: %d", submissionId, resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	viewsStr := strings.TrimSpace(doc.Find(".views .font-large").First().Text())
	views, _ := strconv.Atoi(viewsStr)

	favoritesStr := strings.TrimSpace(doc.Find(".favorites .font-large").First().Text())
	favorites, _ := strconv.Atoi(favoritesStr)

	title := strings.TrimSpace(doc.Find(".submission-title h2 p").First().Text())
    
    descriptionRaw, _ := doc.Find(".submission-description").Html()
    description := common.StripHTMLToText(descriptionRaw)

	content := fmt.Sprintf("%s\n\n%s", title, description)

	dateEl := doc.Find(".popup_date").First()
	
	var postedAt time.Time
	if timestampStr, exists := dateEl.Attr("data-time"); exists {
		if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
			postedAt = time.Unix(timestamp, 0)
		}
	}
	
	if postedAt.IsZero() {
		if dateTitle, exists := dateEl.Attr("title"); exists {
			layout := "Jan 2, 2006 03:04:05 PM"
			postedAt, _ = time.Parse(layout, dateTitle)
		}
	}
	
	if postedAt.IsZero() {
		postedAt = time.Now()
	}

	postID, err := common.CreateOrUpdatePost(
		context.Background(),
		dbQueries,
		sourceId,
		submissionId,
		"FurAffinity",
		postedAt,
		"post",
		username,
		content,
	)
	if err != nil {
		return err
	}

	_, err = dbQueries.SyncReactions(context.Background(), database.SyncReactionsParams{
		ID:       uuid.New(),
		SyncedAt: time.Now(),
		PostID:   postID,
		Likes: sql.NullInt64{
			Int64: int64(favorites),
			Valid: true,
		},
		Reposts: sql.NullInt64{
			Valid: false,
		},
		Views: sql.NullInt64{
			Int64: int64(views),
			Valid: true,
		},
	})

	return err
}
