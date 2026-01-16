package puller

import (
	"context"
	"net/http"

	"github.com/fluffyriot/commission-tracker/internal/auth"
	"github.com/fluffyriot/commission-tracker/internal/database"
	"github.com/google/uuid"
)

func InitializeNoco(tid uuid.UUID, dbQueries *database.Queries, c *Client, encryptionKey []byte, hostUrl string) error {

	target, err := dbQueries.GetTargetById(context.Background(), tid)
	if err != nil {
		return err
	}

	nocoUrl := target.HostUrl.String + "/api/v2/meta/bases/" + target.DbID.String + "/tables"

	req, err := http.NewRequest("POST", nocoUrl, nil)
	if err != nil {
		return err
	}

	err = setNocoHeaders(target.ID, req, dbQueries, encryptionKey)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func setNocoHeaders(tid uuid.UUID, req *http.Request, dbQueries *database.Queries, encryptionKey []byte) error {
	token, _, _, err := auth.GetTargetToken(context.Background(), dbQueries, encryptionKey, tid)
	if err != nil {
		return err
	}
	req.Header.Set(
		"Authorization",
		"Bearer "+token,
	)

	return nil
}
