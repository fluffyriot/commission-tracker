package exports

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/fluffyriot/commission-tracker/internal/database"
	"github.com/google/uuid"
)

func DeleteAllExports(userID uuid.UUID, dbQueries *database.Queries) error {

	exports, err := dbQueries.GetAllExportsByUserId(context.Background(), userID)
	if err != nil {
		log.Printf("Error getting all exports records: %v", err)
		return err
	}
	for _, exp := range exports {
		if exp.DownloadUrl.Valid {
			err := os.Remove(exp.DownloadUrl.String)
			if err != nil {
				log.Printf("Error deleting export file %s: %v", exp.DownloadUrl.String, err)
			}
		}
	}

	err = dbQueries.DeleteAllExportsByUserId(context.Background(), userID)
	if err != nil {
		log.Printf("Error deleting all exports records: %v", err)
		return err
	}

	return nil

}

func InitiateCsvExport(userID uuid.UUID, dbQueries *database.Queries) (database.Export, error) {

	export, err := dbQueries.CreateExport(context.Background(), database.CreateExportParams{
		ID:           uuid.New(),
		CreatedAt:    time.Now(),
		ExportStatus: "Requested",
		UserID:       userID,
		ExportMethod: "csv",
	})

	if err != nil {
		log.Printf("Error creating export record: %v", err)
		return database.Export{}, err
	}

	err = csvExport(userID, dbQueries, export)
	if err != nil {
		return database.Export{}, err
	}

	return export, nil

}

func CreateLogAutoExport(userID uuid.UUID, dbQueries *database.Queries, method string, targetId uuid.UUID) (database.Export, error) {

	export, err := dbQueries.CreateExport(context.Background(), database.CreateExportParams{
		ID:           uuid.New(),
		CreatedAt:    time.Now(),
		ExportStatus: "Requested",
		UserID:       userID,
		ExportMethod: method,
		TargetID:     uuid.NullUUID{UUID: targetId, Valid: true},
	})

	return export, err
}

func UpdateLogAutoExport(export database.Export, dbQueries *database.Queries, status, statusReason string) error {
	var completedDate time.Time
	if status == "Completed" {
		completedDate = time.Now()
	}

	_, err := dbQueries.ChangeExportStatusById(context.Background(), database.ChangeExportStatusByIdParams{
		ID:            export.ID,
		ExportStatus:  status,
		StatusMessage: sql.NullString{String: statusReason, Valid: false},
		CompletedAt:   completedDate,
	})

	return err
}
