// SPDX-License-Identifier: AGPL-3.0-only
package sources

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/fluffyriot/rpsync/internal/authhelp"
	"github.com/fluffyriot/rpsync/internal/database"
	"github.com/google/uuid"
	"golang.org/x/oauth2/google"
	webmasters "google.golang.org/api/webmasters/v3"
	"google.golang.org/api/option"
)

func FetchGoogleSearchConsoleStats(dbQueries *database.Queries, sourceID uuid.UUID, encryptionKey []byte) error {
	ctx := context.Background()

	statsCheck, err := dbQueries.CountAnalyticsSiteStatsBySource(ctx, sourceID)
	if err != nil {
		log.Printf("GSC: Error checking existing stats: %v", err)
	}

	startDate := "7daysAgo"
	if statsCheck == 0 {
		startDate = "730daysAgo"
	}
	endDate := "today"

	return FetchGoogleSearchConsoleStatsWithRange(dbQueries, sourceID, encryptionKey, startDate, endDate)
}

func FetchGoogleSearchConsoleStatsWithRange(dbQueries *database.Queries, sourceID uuid.UUID, encryptionKey []byte, startDate, endDate string) error {
	ctx := context.Background()

	source, err := dbQueries.GetSourceById(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("GSC: failed to get source: %w", err)
	}

	serviceAccountJSON, siteURL, _, _, err := authhelp.GetSourceToken(ctx, dbQueries, encryptionKey, sourceID)
	if err != nil {
		return fmt.Errorf("GSC: failed to get source token: %w", err)
	}

	if siteURL == "" {
		siteURL = source.UserName
	}

	creds, err := google.CredentialsFromJSON(ctx, []byte(serviceAccountJSON), webmasters.WebmastersReadonlyScope)
	if err != nil {
		return fmt.Errorf("GSC: failed to parse credentials: %w", err)
	}

	svc, err := webmasters.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return fmt.Errorf("GSC: failed to create Search Console client: %w", err)
	}

	if err := fetchGSCSiteStats(ctx, svc, dbQueries, sourceID, siteURL, startDate, endDate); err != nil {
		return fmt.Errorf("GSC: failed to fetch site stats: %w", err)
	}

	if err := fetchGSCPageStats(ctx, svc, dbQueries, sourceID, siteURL, startDate, endDate); err != nil {
		return fmt.Errorf("GSC: failed to fetch page stats: %w", err)
	}

	return nil
}

func fetchGSCSiteStats(ctx context.Context, svc *webmasters.Service, db *database.Queries, sourceID uuid.UUID, siteURL, startDate, endDate string) error {
	req := &webmasters.SearchAnalyticsQueryRequest{
		StartDate:  startDate,
		EndDate:    endDate,
		Dimensions: []string{"date"},
		RowLimit:   5000,
	}

	resp, err := svc.Searchanalytics.Query(siteURL, req).Do()
	if err != nil {
		return err
	}

	for _, row := range resp.Rows {
		if len(row.Keys) < 1 {
			continue
		}

		dateStr := row.Keys[0]
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			log.Printf("GSC: Error parsing date %s: %v", dateStr, err)
			continue
		}

		clicks := int64(row.Clicks)
		avgCTR := row.Ctr

		_, err = db.CreateAnalyticsSiteStat(ctx, database.CreateAnalyticsSiteStatParams{
			ID:                 uuid.New(),
			Date:               parsedDate,
			Visitors:           clicks,
			AvgSessionDuration: avgCTR,
			SourceID:           sourceID,
		})
		if err != nil {
			log.Printf("GSC: Error saving site stat for %s: %v", dateStr, err)
		}
	}

	return nil
}

func fetchGSCPageStats(ctx context.Context, svc *webmasters.Service, db *database.Queries, sourceID uuid.UUID, siteURL, startDate, endDate string) error {
	req := &webmasters.SearchAnalyticsQueryRequest{
		StartDate:  startDate,
		EndDate:    endDate,
		Dimensions: []string{"date", "page"},
		RowLimit:   5000,
	}

	resp, err := svc.Searchanalytics.Query(siteURL, req).Do()
	if err != nil {
		return err
	}

	for _, row := range resp.Rows {
		if len(row.Keys) < 2 {
			continue
		}

		dateStr := row.Keys[0]
		pagePath := row.Keys[1]

		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			log.Printf("GSC: Error parsing date %s: %v", dateStr, err)
			continue
		}

		clicks := int64(row.Clicks)

		_, err = db.CreateAnalyticsPageStat(ctx, database.CreateAnalyticsPageStatParams{
			ID:       uuid.New(),
			Date:     parsedDate,
			UrlPath:  pagePath,
			Views:    clicks,
			SourceID: sourceID,
		})
		if err != nil {
			log.Printf("GSC: Error saving page stat for %s %s: %v", dateStr, pagePath, err)
		}
	}

	return nil
}
