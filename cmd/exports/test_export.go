package exports

import (
	"context"
	"database/sql"
	"math/rand"
	"time"

	"github.com/fluffyriot/commission-tracker/internal/database"
	"github.com/google/uuid"
)

func testExport(userID uuid.UUID, dbQueries *database.Queries, exp database.Export) error {

	randomInt := rand.Intn(2)

	var err error

	if randomInt == 0 {
		_, err = dbQueries.ChangeExportStatusById(context.Background(), database.ChangeExportStatusByIdParams{
			ID:            exp.ID,
			ExportStatus:  "Failed",
			StatusMessage: sql.NullString{String: "Random export tester", Valid: true},
			CompletedAt:   time.Now(),
		})
	} else {
		_, err = dbQueries.ChangeExportStatusById(context.Background(), database.ChangeExportStatusByIdParams{
			ID:            exp.ID,
			ExportStatus:  "Succeeded",
			StatusMessage: sql.NullString{},
			CompletedAt:   time.Now(),
		})
	}

	return err
}
