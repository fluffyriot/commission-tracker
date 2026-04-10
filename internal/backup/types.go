// SPDX-License-Identifier: AGPL-3.0-only
package backup

import (
	"encoding/json"
	"time"
)

const ManifestVersion = "1.0"

type Manifest struct {
	Version    string    `json:"version"`
	AppVersion string    `json:"app_version"`
	ExportedAt time.Time `json:"exported_at"`
	Username   string    `json:"username"`
	UserID     string    `json:"user_id"`
}

type BackupUser struct {
	ID              string  `json:"id"`
	Username        string  `json:"username"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
	SyncPeriod      string  `json:"sync_period"`
	ProfileImage    *string `json:"profile_image,omitempty"`
	LastSeenVersion string  `json:"last_seen_version"`
	IntroCompleted  bool    `json:"intro_completed"`
}

type BackupSource struct {
	ID           string  `json:"id"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
	Network      string  `json:"network"`
	UserName     string  `json:"user_name"`
	UserID       string  `json:"user_id"`
	IsActive     bool    `json:"is_active"`
	SyncStatus   string  `json:"sync_status"`
	StatusReason *string `json:"status_reason,omitempty"`
	LastSynced   *string `json:"last_synced,omitempty"`
}

type BackupTarget struct {
	ID            string  `json:"id"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
	TargetType    string  `json:"target_type"`
	UserID        string  `json:"user_id"`
	DbID          *string `json:"db_id,omitempty"`
	IsActive      bool    `json:"is_active"`
	SyncFrequency string  `json:"sync_frequency"`
	SyncStatus    string  `json:"sync_status"`
	StatusReason  *string `json:"status_reason,omitempty"`
	LastSynced    *string `json:"last_synced,omitempty"`
	HostUrl       *string `json:"host_url,omitempty"`
}

type BackupToken struct {
	ID                   string           `json:"id"`
	EncryptedAccessToken string           `json:"encrypted_access_token"`
	Nonce                string           `json:"nonce"`
	CreatedAt            string           `json:"created_at"`
	UpdatedAt            string           `json:"updated_at"`
	ProfileID            *string          `json:"profile_id,omitempty"`
	SourceID             *string          `json:"source_id,omitempty"`
	TargetID             *string          `json:"target_id,omitempty"`
	SourceAppData        *json.RawMessage `json:"source_app_data,omitempty"`
}

type BackupPost struct {
	ID                string  `json:"id"`
	CreatedAt         string  `json:"created_at"`
	LastSyncedAt      string  `json:"last_synced_at"`
	SourceID          string  `json:"source_id"`
	IsArchived        bool    `json:"is_archived"`
	NetworkInternalID string  `json:"network_internal_id"`
	PostType          string  `json:"post_type"`
	Author            string  `json:"author"`
	Content           *string `json:"content,omitempty"`
}

type BackupReaction struct {
	ID       string `json:"id"`
	SyncedAt string `json:"synced_at"`
	PostID   string `json:"post_id"`
	Likes    *int64 `json:"likes,omitempty"`
	Reposts  *int64 `json:"reposts,omitempty"`
	Views    *int64 `json:"views,omitempty"`
}

type BackupTagClassification struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	UserID    string `json:"user_id"`
	Name      string `json:"name"`
}

type BackupTag struct {
	ID               string  `json:"id"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
	UserID           string  `json:"user_id"`
	ClassificationID *string `json:"classification_id,omitempty"`
	Name             string  `json:"name"`
}

type BackupPostTag struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	PostID    string `json:"post_id"`
	TagID     string `json:"tag_id"`
}

type BackupExclusion struct {
	ID                string `json:"id"`
	CreatedAt         string `json:"created_at"`
	SourceID          string `json:"source_id"`
	NetworkInternalID string `json:"network_internal_id"`
}

type BackupRedirect struct {
	ID        string `json:"id"`
	SourceID  string `json:"source_id"`
	FromPath  string `json:"from_path"`
	ToPath    string `json:"to_path"`
	CreatedAt string `json:"created_at"`
}

type BackupSourcesStat struct {
	ID             string   `json:"id"`
	Date           string   `json:"date"`
	SourceID       string   `json:"source_id"`
	FollowersCount *int64   `json:"followers_count,omitempty"`
	FollowingCount *int64   `json:"following_count,omitempty"`
	PostsCount     *int64   `json:"posts_count,omitempty"`
	AverageLikes   *float64 `json:"average_likes,omitempty"`
	AverageReposts *float64 `json:"average_reposts,omitempty"`
	AverageViews   *float64 `json:"average_views,omitempty"`
}

type BackupAnalyticsPageStat struct {
	ID            string `json:"id"`
	Date          string `json:"date"`
	UrlPath       string `json:"url_path"`
	Views         int64  `json:"views"`
	SourceID      string `json:"source_id"`
	AnalyticsType string `json:"analytics_type"`
	Impressions   *int64 `json:"impressions,omitempty"`
}

type BackupAnalyticsSiteStat struct {
	ID                 string  `json:"id"`
	Date               string  `json:"date"`
	Visitors           int64   `json:"visitors"`
	AvgSessionDuration float64 `json:"avg_session_duration"`
	SourceID           string  `json:"source_id"`
	AnalyticsType      string  `json:"analytics_type"`
	Impressions        *int64  `json:"impressions,omitempty"`
}

type ImportResult struct {
	Sources            int    `json:"sources"`
	Targets            int    `json:"targets"`
	Tokens             int    `json:"tokens"`
	Posts              int    `json:"posts"`
	Reactions          int    `json:"reactions"`
	Tags               int    `json:"tags"`
	TagClassifications int    `json:"tag_classifications"`
	PostTags           int    `json:"post_tags"`
	Exclusions         int    `json:"exclusions"`
	Redirects          int    `json:"redirects"`
	SourcesStats       int    `json:"sources_stats"`
	PageStats          int    `json:"page_stats"`
	SiteStats          int    `json:"site_stats"`
	GeneratedUsername  string `json:"generated_username"`
}
