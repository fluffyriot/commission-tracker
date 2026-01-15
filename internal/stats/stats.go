package stats

import (
	"context"
	"time"

	"github.com/fluffyriot/commission-tracker/internal/database"
	"github.com/google/uuid"
)

type ValidationPoint struct {
	Date    time.Time `json:"date"`
	Likes   int64     `json:"likes"`
	Reposts int64     `json:"reposts"`
	Views   int64     `json:"views"`
}

type SourceStats struct {
	SourceID uuid.UUID         `json:"source_id"`
	Network  string            `json:"network"`
	Username string            `json:"username"`
	Points   []ValidationPoint `json:"points"`
}

func GetStats(dbQueries *database.Queries, userID uuid.UUID) ([]SourceStats, error) {

	stats, err := dbQueries.GetDailyStats(context.Background(), userID)
	if err != nil {
		return nil, err
	}

	statsMap := make(map[uuid.UUID]*SourceStats)

	for _, row := range stats {

		if _, ok := statsMap[row.ID]; !ok {
			statsMap[row.ID] = &SourceStats{
				SourceID: row.ID,
				Network:  row.Network,
				Username: row.UserName,
				Points:   []ValidationPoint{},
			}
		}

		statsMap[row.ID].Points = append(statsMap[row.ID].Points, ValidationPoint{
			Date:    row.Date,
			Likes:   row.TotalLikes,
			Reposts: row.TotalReposts,
			Views:   row.TotalViews,
		})
	}

	var result []SourceStats
	for _, s := range statsMap {
		result = append(result, *s)
	}

	return result, nil
}
