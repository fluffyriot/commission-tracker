// SPDX-License-Identifier: AGPL-3.0-only
package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/fluffyriot/rpsync/internal/database"
	"github.com/fluffyriot/rpsync/internal/helpers"
	"github.com/fluffyriot/rpsync/internal/stats"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type analyticsFilters struct {
	UserID    uuid.UUID
	StartDate sql.NullTime
	EndDate   sql.NullTime
	PostTypes []string
	HasFilter bool
}

func parseAnalyticsFilters(c *gin.Context, userID uuid.UUID) analyticsFilters {
	f := analyticsFilters{UserID: userID}

	if sd := c.Query("start_date"); sd != "" {
		if t, err := time.Parse("2006-01-02", sd); err == nil {
			f.StartDate = sql.NullTime{Time: t, Valid: true}
			f.HasFilter = true
		}
	}
	if ed := c.Query("end_date"); ed != "" {
		if t, err := time.Parse("2006-01-02", ed); err == nil {
			f.EndDate = sql.NullTime{Time: t, Valid: true}
			f.HasFilter = true
		}
	}
	if pt := c.Query("post_types"); pt != "" {
		f.PostTypes = strings.Split(pt, ",")
		f.HasFilter = true
	}

	return f
}

func (h *Handler) AnalyticsEngagementHandler(c *gin.Context) {

	if h.Config.DBInitErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": h.Config.DBInitErr.Error()})
		return
	}

	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	statsData, err := stats.GetStats(h.DB, user.ID)
	if err != nil {
		log.Printf("Error getting stats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, statsData)
}

func (h *Handler) AnalyticsWebsiteHandler(c *gin.Context) {
	if h.Config.DBInitErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": h.Config.DBInitErr.Error()})
		return
	}

	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	statsData, err := stats.GetAnalyticsStats(h.DB, user.ID)
	if err != nil {
		log.Printf("Error getting analytics stats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, statsData)
}

func (h *Handler) AnalyticsDashboardSummaryHandler(c *gin.Context) {
	if h.Config.DBInitErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": h.Config.DBInitErr.Error()})
		return
	}

	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	summary, err := stats.GetDashboardSummary(h.DB, user.ID)
	if err != nil {
		log.Printf("Error getting dashboard summary: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

func (h *Handler) AnalyticsTopSourcesHandler(c *gin.Context) {
	if h.Config.DBInitErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": h.Config.DBInitErr.Error()})
		return
	}

	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	topSourcesDB, err := h.DB.GetRestTopSources(c.Request.Context(), user.ID)
	if err != nil {
		log.Printf("Error getting rest of top sources: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var topSources []TopSourceViewModel
	for _, src := range topSourcesDB {
		profileURL, _ := helpers.ConvNetworkToURL(src.Network, src.UserName)
		topSources = append(topSources, TopSourceViewModel{
			ID:                src.ID,
			UserName:          src.UserName,
			Network:           src.Network,
			TotalInteractions: int64(src.TotalInteractions),
			FollowersCount:    int64(src.FollowersCount),
			ProfileURL:        profileURL,
		})
	}

	if topSources == nil {
		topSources = []TopSourceViewModel{}
	}

	c.JSON(http.StatusOK, topSources)
}

func (h *Handler) AnalyticsHandler(c *gin.Context) {
	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	c.HTML(http.StatusOK, "analytics.html", h.CommonData(c, gin.H{
		"title":   "Advanced Analytics",
		"user_id": user.ID,
	}))
}

func (h *Handler) AnalyticsWordCloudHandler(c *gin.Context) {
	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	f := parseAnalyticsFilters(c, user.ID)
	var data interface{}
	var err error
	if f.HasFilter {
		data, err = h.DB.GetWordCloudDataFiltered(c.Request.Context(), database.GetWordCloudDataFilteredParams{
			UserID:    f.UserID,
			StartDate: f.StartDate,
			EndDate:   f.EndDate,
			PostTypes: f.PostTypes,
		})
	} else {
		data, err = h.DB.GetWordCloudData(c.Request.Context(), user.ID)
	}
	if err != nil {
		log.Printf("Error getting word cloud data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *Handler) AnalyticsHashtagsHandler(c *gin.Context) {
	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	f := parseAnalyticsFilters(c, user.ID)
	var data interface{}
	var err error
	if f.HasFilter {
		data, err = h.DB.GetHashtagAnalyticsFiltered(c.Request.Context(), database.GetHashtagAnalyticsFilteredParams{
			UserID:    f.UserID,
			StartDate: f.StartDate,
			EndDate:   f.EndDate,
			PostTypes: f.PostTypes,
		})
	} else {
		data, err = h.DB.GetHashtagAnalytics(c.Request.Context(), user.ID)
	}
	if err != nil {
		log.Printf("Error getting hashtags data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *Handler) AnalyticsMentionsHandler(c *gin.Context) {
	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	f := parseAnalyticsFilters(c, user.ID)
	var data interface{}
	var err error
	if f.HasFilter {
		data, err = h.DB.GetMentionsAnalyticsFiltered(c.Request.Context(), database.GetMentionsAnalyticsFilteredParams{
			UserID:    f.UserID,
			StartDate: f.StartDate,
			EndDate:   f.EndDate,
			PostTypes: f.PostTypes,
		})
	} else {
		data, err = h.DB.GetMentionsAnalytics(c.Request.Context(), user.ID)
	}
	if err != nil {
		log.Printf("Error getting mentions data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *Handler) AnalyticsTimeHandler(c *gin.Context) {
	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	f := parseAnalyticsFilters(c, user.ID)
	var data interface{}
	var err error
	if f.HasFilter {
		data, err = h.DB.GetTimePerformanceFiltered(c.Request.Context(), database.GetTimePerformanceFilteredParams{
			UserID:    f.UserID,
			StartDate: f.StartDate,
			EndDate:   f.EndDate,
			PostTypes: f.PostTypes,
		})
	} else {
		data, err = h.DB.GetTimePerformance(c.Request.Context(), user.ID)
	}
	if err != nil {
		log.Printf("Error getting time performance data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *Handler) AnalyticsPostTypesHandler(c *gin.Context) {
	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	f := parseAnalyticsFilters(c, user.ID)
	var data interface{}
	var err error
	if f.HasFilter {
		data, err = h.DB.GetGlobalPostTypeAnalyticsFiltered(c.Request.Context(), database.GetGlobalPostTypeAnalyticsFilteredParams{
			UserID:    f.UserID,
			StartDate: f.StartDate,
			EndDate:   f.EndDate,
			PostTypes: f.PostTypes,
		})
	} else {
		data, err = h.DB.GetGlobalPostTypeAnalytics(c.Request.Context(), user.ID)
	}
	if err != nil {
		log.Printf("Error getting post type analytics: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *Handler) AnalyticsNetworkEfficiencyHandler(c *gin.Context) {
	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	f := parseAnalyticsFilters(c, user.ID)
	var data interface{}
	var err error
	if f.HasFilter {
		data, err = h.DB.GetNetworkEfficiencyFiltered(c.Request.Context(), database.GetNetworkEfficiencyFilteredParams{
			UserID:    f.UserID,
			StartDate: f.StartDate,
			EndDate:   f.EndDate,
			PostTypes: f.PostTypes,
		})
	} else {
		data, err = h.DB.GetNetworkEfficiency(c.Request.Context(), user.ID)
	}
	if err != nil {
		log.Printf("Error getting network efficiency: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *Handler) AnalyticsTopPagesHandler(c *gin.Context) {
	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	f := parseAnalyticsFilters(c, user.ID)
	var data interface{}
	var err error
	if f.HasFilter && (f.StartDate.Valid || f.EndDate.Valid) {
		data, err = h.DB.GetTopPagesByViewsFiltered(c.Request.Context(), database.GetTopPagesByViewsFilteredParams{
			UserID:    f.UserID,
			StartDate: f.StartDate,
			EndDate:   f.EndDate,
		})
	} else {
		data, err = h.DB.GetTopPagesByViews(c.Request.Context(), user.ID)
	}
	if err != nil {
		log.Printf("Error getting top pages: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *Handler) AnalyticsSiteStatsHandler(c *gin.Context) {
	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	f := parseAnalyticsFilters(c, user.ID)
	var data interface{}
	var err error
	if f.HasFilter && (f.StartDate.Valid || f.EndDate.Valid) {
		data, err = h.DB.GetSiteStatsOverTimeFiltered(c.Request.Context(), database.GetSiteStatsOverTimeFilteredParams{
			UserID:    f.UserID,
			StartDate: f.StartDate,
			EndDate:   f.EndDate,
		})
	} else {
		data, err = h.DB.GetSiteStatsOverTime(c.Request.Context(), user.ID)
	}
	if err != nil {
		log.Printf("Error getting site stats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *Handler) AnalyticsPostingConsistencyHandler(c *gin.Context) {
	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	f := parseAnalyticsFilters(c, user.ID)
	var data interface{}
	var err error
	if f.HasFilter {
		data, err = h.DB.GetPostingConsistencyFiltered(c.Request.Context(), database.GetPostingConsistencyFilteredParams{
			UserID:    f.UserID,
			StartDate: f.StartDate,
			EndDate:   f.EndDate,
			PostTypes: f.PostTypes,
		})
	} else {
		data, err = h.DB.GetPostingConsistency(c.Request.Context(), user.ID)
	}
	if err != nil {
		log.Printf("Error getting posting consistency: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *Handler) AnalyticsEngagementRateHandler(c *gin.Context) {
	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	f := parseAnalyticsFilters(c, user.ID)
	var data interface{}
	var err error
	if f.HasFilter {
		data, err = h.DB.GetEngagementRateDataFiltered(c.Request.Context(), database.GetEngagementRateDataFilteredParams{
			UserID:    f.UserID,
			StartDate: f.StartDate,
			EndDate:   f.EndDate,
			PostTypes: f.PostTypes,
		})
	} else {
		data, err = h.DB.GetEngagementRateData(c.Request.Context(), user.ID)
	}
	if err != nil {
		log.Printf("Error getting engagement rate data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *Handler) AnalyticsFollowRatioHandler(c *gin.Context) {
	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	data, err := h.DB.GetFollowRatioData(c.Request.Context(), user.ID)
	if err != nil {
		log.Printf("Error getting follow ratio data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if data == nil {
		data = []database.GetFollowRatioDataRow{}
	}
	c.JSON(http.StatusOK, data)
}

func (h *Handler) AnalyticsPerformanceDeviationHandler(c *gin.Context) {
	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	f := parseAnalyticsFilters(c, user.ID)

	var positiveData []database.GetPerformanceDeviationPositiveRow
	var negativeData []database.GetPerformanceDeviationNegativeRow
	var err error

	if f.HasFilter {
		rawPos, e := h.DB.GetPerformanceDeviationPositiveFiltered(c.Request.Context(), database.GetPerformanceDeviationPositiveFilteredParams{
			UserID:    f.UserID,
			StartDate: f.StartDate,
			EndDate:   f.EndDate,
			PostTypes: f.PostTypes,
		})
		if e != nil {
			log.Printf("Error getting performance deviation positive data: %v", e)
			c.JSON(http.StatusInternalServerError, gin.H{"error": e.Error()})
			return
		}
		for _, r := range rawPos {
			positiveData = append(positiveData, database.GetPerformanceDeviationPositiveRow{
				ID:                 r.ID,
				NetworkInternalID:  r.NetworkInternalID,
				Content:            r.Content,
				CreatedAt:          r.CreatedAt,
				Author:             r.Author,
				Network:            r.Network,
				Likes:              r.Likes,
				Reposts:            r.Reposts,
				ExpectedEngagement: r.ExpectedEngagement,
			})
		}
		rawNeg, e := h.DB.GetPerformanceDeviationNegativeFiltered(c.Request.Context(), database.GetPerformanceDeviationNegativeFilteredParams{
			UserID:    f.UserID,
			StartDate: f.StartDate,
			EndDate:   f.EndDate,
			PostTypes: f.PostTypes,
		})
		if e != nil {
			log.Printf("Error getting performance deviation negative data: %v", e)
			c.JSON(http.StatusInternalServerError, gin.H{"error": e.Error()})
			return
		}
		for _, r := range rawNeg {
			negativeData = append(negativeData, database.GetPerformanceDeviationNegativeRow{
				ID:                 r.ID,
				NetworkInternalID:  r.NetworkInternalID,
				Content:            r.Content,
				CreatedAt:          r.CreatedAt,
				Author:             r.Author,
				Network:            r.Network,
				Likes:              r.Likes,
				Reposts:            r.Reposts,
				ExpectedEngagement: r.ExpectedEngagement,
			})
		}
	} else {
		positiveData, err = h.DB.GetPerformanceDeviationPositive(c.Request.Context(), user.ID)
		if err != nil {
			log.Printf("Error getting performance deviation positive data: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		negativeData, err = h.DB.GetPerformanceDeviationNegative(c.Request.Context(), user.ID)
		if err != nil {
			log.Printf("Error getting performance deviation negative data: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	type DeviationItem struct {
		ID                 interface{} `json:"id"`
		NetworkInternalID  string      `json:"network_internal_id"`
		Content            string      `json:"content"`
		CreatedAt          interface{} `json:"created_at"`
		Author             string      `json:"author"`
		Network            string      `json:"network"`
		Likes              int64       `json:"likes"`
		Reposts            int64       `json:"reposts"`
		ExpectedEngagement float64     `json:"expected_engagement"`
		URL                string      `json:"url"`
		Deviation          float64     `json:"deviation"`
	}

	var positive []DeviationItem
	for _, item := range positiveData {
		url := ""
		if item.Network != "" && item.Author != "" {
			url, _ = helpers.ConvPostToURL(item.Network, item.Author, item.NetworkInternalID)
		}
		positive = append(positive, DeviationItem{
			ID:                 item.ID,
			NetworkInternalID:  item.NetworkInternalID,
			Content:            item.Content,
			CreatedAt:          item.CreatedAt,
			Author:             item.Author,
			Network:            item.Network,
			Likes:              item.Likes,
			Reposts:            item.Reposts,
			ExpectedEngagement: item.ExpectedEngagement,
			URL:                url,
			Deviation:          float64(item.Likes+item.Reposts) - item.ExpectedEngagement,
		})
	}

	var negative []DeviationItem
	for _, item := range negativeData {
		url := ""
		if item.Network != "" && item.Author != "" {
			url, _ = helpers.ConvPostToURL(item.Network, item.Author, item.NetworkInternalID)
		}
		negative = append(negative, DeviationItem{
			ID:                 item.ID,
			NetworkInternalID:  item.NetworkInternalID,
			Content:            item.Content,
			CreatedAt:          item.CreatedAt,
			Author:             item.Author,
			Network:            item.Network,
			Likes:              item.Likes,
			Reposts:            item.Reposts,
			ExpectedEngagement: item.ExpectedEngagement,
			URL:                url,
			Deviation:          float64(item.Likes+item.Reposts) - item.ExpectedEngagement,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"positive": positive,
		"negative": negative,
	})
}

func (h *Handler) AnalyticsVelocityHandler(c *gin.Context) {
	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	f := parseAnalyticsFilters(c, user.ID)

	type VelocityItem struct {
		PostID          interface{} `json:"post_id"`
		HistorySyncedAt interface{} `json:"history_synced_at"`
		Likes           int64       `json:"likes"`
		Reposts         int64       `json:"reposts"`
		PostCreatedAt   interface{} `json:"post_created_at"`
		Content         string      `json:"content"`
		Author          string      `json:"author"`
		NetworkInternalID string    `json:"network_internal_id"`
		Network         string      `json:"network"`
		URL             string      `json:"url"`
	}

	var items []VelocityItem

	if f.HasFilter {
		data, err := h.DB.GetEngagementVelocityDataFiltered(c.Request.Context(), database.GetEngagementVelocityDataFilteredParams{
			UserID:    f.UserID,
			StartDate: f.StartDate,
			EndDate:   f.EndDate,
			PostTypes: f.PostTypes,
		})
		if err != nil {
			log.Printf("Error getting engagement velocity data: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		items = make([]VelocityItem, len(data))
		for i, d := range data {
			url := ""
			if d.Network != "" && d.Author != "" {
				url, _ = helpers.ConvPostToURL(d.Network, d.Author, d.NetworkInternalID)
			}
			items[i] = VelocityItem{d.PostID, d.HistorySyncedAt, d.Likes, d.Reposts, d.PostCreatedAt, d.Content, d.Author, d.NetworkInternalID, d.Network, url}
		}
	} else {
		data, err := h.DB.GetEngagementVelocityData(c.Request.Context(), user.ID)
		if err != nil {
			log.Printf("Error getting engagement velocity data: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		items = make([]VelocityItem, len(data))
		for i, d := range data {
			url := ""
			if d.Network != "" && d.Author != "" {
				url, _ = helpers.ConvPostToURL(d.Network, d.Author, d.NetworkInternalID)
			}
			items[i] = VelocityItem{d.PostID, d.HistorySyncedAt, d.Likes, d.Reposts, d.PostCreatedAt, d.Content, d.Author, d.NetworkInternalID, d.Network, url}
		}
	}

	c.JSON(http.StatusOK, items)
}

func (h *Handler) AnalyticsCollaborationsHandler(c *gin.Context) {
	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	f := parseAnalyticsFilters(c, user.ID)
	var data interface{}
	var err error
	if f.HasFilter {
		data, err = h.DB.GetCollaborationsDataFiltered(c.Request.Context(), database.GetCollaborationsDataFilteredParams{
			UserID:    f.UserID,
			StartDate: f.StartDate,
			EndDate:   f.EndDate,
		})
	} else {
		data, err = h.DB.GetCollaborationsData(c.Request.Context(), user.ID)
	}
	if err != nil {
		log.Printf("Error getting collaborations data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *Handler) AnalyticsWordCloudEngagementHandler(c *gin.Context) {
	user, loggedIn := h.GetAuthenticatedUser(c)
	if !loggedIn {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	f := parseAnalyticsFilters(c, user.ID)
	var data interface{}
	var err error
	if f.HasFilter {
		filteredData, e := h.DB.GetWordCloudEngagementDataFiltered(c.Request.Context(), database.GetWordCloudEngagementDataFilteredParams{
			UserID:    f.UserID,
			StartDate: f.StartDate,
			EndDate:   f.EndDate,
			PostTypes: f.PostTypes,
		})
		if filteredData == nil {
			filteredData = []database.GetWordCloudEngagementDataFilteredRow{}
		}
		data, err = filteredData, e
	} else {
		rawData, e := h.DB.GetWordCloudEngagementData(c.Request.Context(), user.ID)
		if rawData == nil {
			rawData = []database.GetWordCloudEngagementDataRow{}
		}
		data, err = rawData, e
	}
	if err != nil {
		log.Printf("Error getting word cloud engagement data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}
