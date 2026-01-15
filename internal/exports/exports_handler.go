package exports

import (
	"context"
	"errors"
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

func InitiateExport(userID uuid.UUID, syncMethod string, dbQueries *database.Queries) (database.Export, error) {

	export, err := dbQueries.CreateExport(context.Background(), database.CreateExportParams{
		ID:           uuid.New(),
		CreatedAt:    time.Now(),
		ExportStatus: "Requested",
		UserID:       userID,
		ExportMethod: syncMethod,
	})

	if err != nil {
		log.Printf("Error creating export record: %v", err)
		return database.Export{}, err
	}

	switch syncMethod {
	case "csv":

		err = csvExport(userID, dbQueries, export)
		if err != nil {
			return database.Export{}, err
		}

		return export, nil

	case "notion":

		err = notionExport(userID, dbQueries, export)
		if err != nil {
			return database.Export{}, err
		}

		return export, nil

	case "none":

		err = testExport(userID, dbQueries, export)
		if err != nil {
			return database.Export{}, err
		}

		return export, nil

	default:
		return database.Export{}, errors.New("Unknown sync method")
	}

}
