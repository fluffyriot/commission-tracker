package exports

import (
	"context"
	"database/sql"
	"time"

	"github.com/fluffyriot/commission-tracker/internal/database"
	"github.com/google/uuid"
)

func notionExport(userID uuid.UUID, dbQueries *database.Queries, exp database.Export) error {

	_, err := dbQueries.ChangeExportStatusById(context.Background(), database.ChangeExportStatusByIdParams{
		ID:            exp.ID,
		ExportStatus:  "Failed",
		StatusMessage: sql.NullString{String: "Not implemented yet", Valid: true},
		CompletedAt:   time.Now(),
	})

	return err
}
