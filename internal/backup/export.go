// SPDX-License-Identifier: AGPL-3.0-only
package backup

import (
	"archive/zip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fluffyriot/rpsync/internal/config"
	"github.com/fluffyriot/rpsync/internal/database"
	"github.com/google/uuid"
)

const timeFormat = time.RFC3339

func ExportUserData(ctx context.Context, db *database.Queries, userID uuid.UUID, exportID uuid.UUID) (string, error) {
	user, err := db.BackupGetUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	manifest := Manifest{
		Version:    ManifestVersion,
		AppVersion: config.AppVersion,
		ExportedAt: time.Now().UTC(),
		Username:   user.Username,
		UserID:     user.ID.String(),
	}

	filename := fmt.Sprintf("export_id_%s_backup.zip", exportID.String())
	outputDir, err := filepath.Abs("./outputs")
	if err != nil {
		return "", fmt.Errorf("failed to resolve output dir: %w", err)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output dir: %w", err)
	}
	fullPath := filepath.Join(outputDir, filename)

	f, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create zip file: %w", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	if err := writeJSON(w, "manifest.json", manifest); err != nil {
		return "", err
	}

	backupUser := convertUser(user)
	if err := writeJSON(w, "user.json", backupUser); err != nil {
		return "", err
	}

	sources, err := db.BackupGetSourcesForUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get sources: %w", err)
	}
	if err := writeJSON(w, "sources.json", convertSources(sources)); err != nil {
		return "", err
	}

	targets, err := db.BackupGetTargetsForUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get targets: %w", err)
	}
	if err := writeJSON(w, "targets.json", convertTargets(targets)); err != nil {
		return "", err
	}

	tokens, err := db.BackupGetTokensForUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get tokens: %w", err)
	}
	if err := writeJSON(w, "tokens.json", convertTokens(tokens)); err != nil {
		return "", err
	}

	posts, err := db.BackupGetPostsForUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get posts: %w", err)
	}
	if err := writeJSON(w, "posts.json", convertPosts(posts)); err != nil {
		return "", err
	}

	reactions, err := db.BackupGetReactionsForUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get reactions: %w", err)
	}
	if err := writeJSON(w, "posts_reactions_history.json", convertReactions(reactions)); err != nil {
		return "", err
	}

	tagClassifications, err := db.BackupGetTagClassificationsForUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get tag classifications: %w", err)
	}
	if err := writeJSON(w, "tag_classifications.json", convertTagClassifications(tagClassifications)); err != nil {
		return "", err
	}

	tags, err := db.BackupGetTagsForUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get tags: %w", err)
	}
	if err := writeJSON(w, "tags.json", convertTags(tags)); err != nil {
		return "", err
	}

	postTags, err := db.BackupGetPostTagsForUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get post tags: %w", err)
	}
	if err := writeJSON(w, "post_tags.json", convertPostTags(postTags)); err != nil {
		return "", err
	}

	exclusions, err := db.BackupGetExclusionsForUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get exclusions: %w", err)
	}
	if err := writeJSON(w, "exclusions.json", convertExclusions(exclusions)); err != nil {
		return "", err
	}

	redirects, err := db.BackupGetRedirectsForUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get redirects: %w", err)
	}
	if err := writeJSON(w, "redirects.json", convertRedirects(redirects)); err != nil {
		return "", err
	}

	sourcesStats, err := db.BackupGetSourcesStatsForUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get sources stats: %w", err)
	}
	if err := writeJSON(w, "sources_stats.json", convertSourcesStats(sourcesStats)); err != nil {
		return "", err
	}

	pageStats, err := db.BackupGetAnalyticsPageStatsForUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get page stats: %w", err)
	}
	if err := writeJSON(w, "analytics_page_stats.json", convertPageStats(pageStats)); err != nil {
		return "", err
	}

	siteStats, err := db.BackupGetAnalyticsSiteStatsForUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get site stats: %w", err)
	}
	if err := writeJSON(w, "analytics_site_stats.json", convertSiteStats(siteStats)); err != nil {
		return "", err
	}

	return fullPath, nil
}

func writeJSON(w *zip.Writer, name string, data any) error {
	fw, err := w.Create(name)
	if err != nil {
		return fmt.Errorf("failed to create zip entry %s: %w", name, err)
	}
	enc := json.NewEncoder(fw)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("failed to encode %s: %w", name, err)
	}
	return nil
}

func convertUser(u database.BackupGetUserRow) BackupUser {
	bu := BackupUser{
		ID:              u.ID.String(),
		Username:        u.Username,
		CreatedAt:       u.CreatedAt.Format(timeFormat),
		UpdatedAt:       u.UpdatedAt.Format(timeFormat),
		SyncPeriod:      u.SyncPeriod,
		LastSeenVersion: u.LastSeenVersion,
		IntroCompleted:  u.IntroCompleted,
	}
	if u.ProfileImage.Valid {
		bu.ProfileImage = &u.ProfileImage.String
	}
	return bu
}

func convertSources(sources []database.Source) []BackupSource {
	result := make([]BackupSource, 0, len(sources))
	for _, s := range sources {
		bs := BackupSource{
			ID:         s.ID.String(),
			CreatedAt:  s.CreatedAt.Format(timeFormat),
			UpdatedAt:  s.UpdatedAt.Format(timeFormat),
			Network:    s.Network,
			UserName:   s.UserName,
			UserID:     s.UserID.String(),
			IsActive:   s.IsActive,
			SyncStatus: s.SyncStatus,
		}
		if s.StatusReason.Valid {
			bs.StatusReason = &s.StatusReason.String
		}
		if s.LastSynced.Valid {
			t := s.LastSynced.Time.Format(timeFormat)
			bs.LastSynced = &t
		}
		result = append(result, bs)
	}
	return result
}

func convertTargets(targets []database.Target) []BackupTarget {
	result := make([]BackupTarget, 0, len(targets))
	for _, t := range targets {
		bt := BackupTarget{
			ID:            t.ID.String(),
			CreatedAt:     t.CreatedAt.Format(timeFormat),
			UpdatedAt:     t.UpdatedAt.Format(timeFormat),
			TargetType:    t.TargetType,
			UserID:        t.UserID.String(),
			IsActive:      t.IsActive,
			SyncFrequency: t.SyncFrequency,
			SyncStatus:    t.SyncStatus,
		}
		if t.DbID.Valid {
			bt.DbID = &t.DbID.String
		}
		if t.StatusReason.Valid {
			bt.StatusReason = &t.StatusReason.String
		}
		if t.LastSynced.Valid {
			ts := t.LastSynced.Time.Format(timeFormat)
			bt.LastSynced = &ts
		}
		if t.HostUrl.Valid {
			bt.HostUrl = &t.HostUrl.String
		}
		result = append(result, bt)
	}
	return result
}

func convertTokens(tokens []database.Token) []BackupToken {
	result := make([]BackupToken, 0, len(tokens))
	for _, t := range tokens {
		bt := BackupToken{
			ID:                   t.ID.String(),
			EncryptedAccessToken: base64.StdEncoding.EncodeToString(t.EncryptedAccessToken),
			Nonce:                base64.StdEncoding.EncodeToString(t.Nonce),
			CreatedAt:            t.CreatedAt.Format(timeFormat),
			UpdatedAt:            t.UpdatedAt.Format(timeFormat),
		}
		if t.ProfileID.Valid {
			bt.ProfileID = &t.ProfileID.String
		}
		if t.SourceID.Valid {
			s := t.SourceID.UUID.String()
			bt.SourceID = &s
		}
		if t.TargetID.Valid {
			s := t.TargetID.UUID.String()
			bt.TargetID = &s
		}
		if t.SourceAppData != nil {
			raw := json.RawMessage(t.SourceAppData)
			bt.SourceAppData = &raw
		}
		result = append(result, bt)
	}
	return result
}

func convertPosts(posts []database.Post) []BackupPost {
	result := make([]BackupPost, 0, len(posts))
	for _, p := range posts {
		bp := BackupPost{
			ID:                p.ID.String(),
			CreatedAt:         p.CreatedAt.Format(timeFormat),
			LastSyncedAt:      p.LastSyncedAt.Format(timeFormat),
			SourceID:          p.SourceID.String(),
			IsArchived:        p.IsArchived,
			NetworkInternalID: p.NetworkInternalID,
			PostType:          p.PostType,
			Author:            p.Author,
		}
		if p.Content.Valid {
			bp.Content = &p.Content.String
		}
		result = append(result, bp)
	}
	return result
}

func convertReactions(reactions []database.PostsReactionsHistory) []BackupReaction {
	result := make([]BackupReaction, 0, len(reactions))
	for _, r := range reactions {
		br := BackupReaction{
			ID:       r.ID.String(),
			SyncedAt: r.SyncedAt.Format(timeFormat),
			PostID:   r.PostID.String(),
		}
		if r.Likes.Valid {
			br.Likes = &r.Likes.Int64
		}
		if r.Reposts.Valid {
			br.Reposts = &r.Reposts.Int64
		}
		if r.Views.Valid {
			br.Views = &r.Views.Int64
		}
		result = append(result, br)
	}
	return result
}

func convertTagClassifications(tcs []database.TagClassification) []BackupTagClassification {
	result := make([]BackupTagClassification, 0, len(tcs))
	for _, tc := range tcs {
		result = append(result, BackupTagClassification{
			ID:        tc.ID.String(),
			CreatedAt: tc.CreatedAt.Format(timeFormat),
			UpdatedAt: tc.UpdatedAt.Format(timeFormat),
			UserID:    tc.UserID.String(),
			Name:      tc.Name,
		})
	}
	return result
}

func convertTags(tags []database.Tag) []BackupTag {
	result := make([]BackupTag, 0, len(tags))
	for _, t := range tags {
		bt := BackupTag{
			ID:        t.ID.String(),
			CreatedAt: t.CreatedAt.Format(timeFormat),
			UpdatedAt: t.UpdatedAt.Format(timeFormat),
			UserID:    t.UserID.String(),
			Name:      t.Name,
		}
		if t.ClassificationID.Valid {
			s := t.ClassificationID.UUID.String()
			bt.ClassificationID = &s
		}
		result = append(result, bt)
	}
	return result
}

func convertPostTags(postTags []database.PostTag) []BackupPostTag {
	result := make([]BackupPostTag, 0, len(postTags))
	for _, pt := range postTags {
		result = append(result, BackupPostTag{
			ID:        pt.ID.String(),
			CreatedAt: pt.CreatedAt.Format(timeFormat),
			PostID:    pt.PostID.String(),
			TagID:     pt.TagID.String(),
		})
	}
	return result
}

func convertExclusions(exclusions []database.Exclusion) []BackupExclusion {
	result := make([]BackupExclusion, 0, len(exclusions))
	for _, e := range exclusions {
		result = append(result, BackupExclusion{
			ID:                e.ID.String(),
			CreatedAt:         e.CreatedAt.Format(timeFormat),
			SourceID:          e.SourceID.String(),
			NetworkInternalID: e.NetworkInternalID,
		})
	}
	return result
}

func convertRedirects(redirects []database.Redirect) []BackupRedirect {
	result := make([]BackupRedirect, 0, len(redirects))
	for _, r := range redirects {
		result = append(result, BackupRedirect{
			ID:        r.ID.String(),
			SourceID:  r.SourceID.String(),
			FromPath:  r.FromPath,
			ToPath:    r.ToPath,
			CreatedAt: r.CreatedAt.Format(timeFormat),
		})
	}
	return result
}

func convertSourcesStats(stats []database.SourcesStat) []BackupSourcesStat {
	result := make([]BackupSourcesStat, 0, len(stats))
	for _, s := range stats {
		bs := BackupSourcesStat{
			ID:       s.ID.String(),
			Date:     s.Date.Format(timeFormat),
			SourceID: s.SourceID.String(),
		}
		if s.FollowersCount.Valid {
			bs.FollowersCount = &s.FollowersCount.Int64
		}
		if s.FollowingCount.Valid {
			bs.FollowingCount = &s.FollowingCount.Int64
		}
		if s.PostsCount.Valid {
			bs.PostsCount = &s.PostsCount.Int64
		}
		if s.AverageLikes.Valid {
			bs.AverageLikes = &s.AverageLikes.Float64
		}
		if s.AverageReposts.Valid {
			bs.AverageReposts = &s.AverageReposts.Float64
		}
		if s.AverageViews.Valid {
			bs.AverageViews = &s.AverageViews.Float64
		}
		result = append(result, bs)
	}
	return result
}

func convertPageStats(stats []database.AnalyticsPageStat) []BackupAnalyticsPageStat {
	result := make([]BackupAnalyticsPageStat, 0, len(stats))
	for _, s := range stats {
		bs := BackupAnalyticsPageStat{
			ID:            s.ID.String(),
			Date:          s.Date.Format(timeFormat),
			UrlPath:       s.UrlPath,
			Views:         s.Views,
			SourceID:      s.SourceID.String(),
			AnalyticsType: s.AnalyticsType,
		}
		if s.Impressions.Valid {
			bs.Impressions = &s.Impressions.Int64
		}
		result = append(result, bs)
	}
	return result
}

func convertSiteStats(stats []database.AnalyticsSiteStat) []BackupAnalyticsSiteStat {
	result := make([]BackupAnalyticsSiteStat, 0, len(stats))
	for _, s := range stats {
		bs := BackupAnalyticsSiteStat{
			ID:                 s.ID.String(),
			Date:               s.Date.Format(timeFormat),
			Visitors:           s.Visitors,
			AvgSessionDuration: s.AvgSessionDuration,
			SourceID:           s.SourceID.String(),
			AnalyticsType:      s.AnalyticsType,
		}
		if s.Impressions.Valid {
			bs.Impressions = &s.Impressions.Int64
		}
		result = append(result, bs)
	}
	return result
}
