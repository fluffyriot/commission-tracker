// SPDX-License-Identifier: AGPL-3.0-only
package handlers

import "github.com/google/uuid"

type TopSourceViewModel struct {
	ID                  uuid.UUID `json:"id"`
	UserName            string    `json:"username"`
	Network             string    `json:"network"`
	TotalInteractions   int64     `json:"total_interactions"`
	TotalViews          int64     `json:"total_views"`
	FollowersCount      int64     `json:"followers_count"`
	ProfileURL          string    `json:"profile_url"`
	EngagementSupported bool      `json:"engagement_supported"`
	ViewsSupported      bool      `json:"views_supported"`
	FollowersTracked    bool      `json:"followers_tracked"`
}

type DashboardLogItem struct {
	ID             string `json:"id"`
	CreatedAt      string `json:"created_at"`
	SourceNetwork  string `json:"source_network"`
	SourceUsername string `json:"source_username"`
	TargetType     string `json:"target_type"`
	Message        string `json:"message"`
}

type DashboardStatsResponse struct {
	ActiveSources         int64                `json:"active_sources"`
	ActiveTargets         int64                `json:"active_targets"`
	TotalPosts            int64                `json:"total_posts"`
	TotalLikes            int64                `json:"total_likes"`
	TotalShares           int64                `json:"total_shares"`
	TotalViews            int64                `json:"total_views"`
	TotalVisitors         int64                `json:"total_visitors"`
	TotalPageViews        int64                `json:"total_page_views"`
	AverageWebsiteSession int64                `json:"average_website_session"`
	SyncErrors30d         int64                `json:"sync_errors_30d"`
	TopSources            []TopSourceViewModel `json:"top_sources"`
}
