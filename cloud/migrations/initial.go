// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrations

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func Register(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if err := ensureUserFields(app); err != nil {
			return err
		}
		if err := ensureDownloadTokensCollection(app); err != nil {
			return err
		}
		return se.Next()
	})
}

func ensureUserFields(app *pocketbase.PocketBase) error {
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return err
	}
	if users.Fields.GetByName("company") == nil {
		users.Fields.Add(&core.TextField{
			Name:     "company",
			Required: false,
		})
		if err := app.Save(users); err != nil {
			return err
		}
	}
	return nil
}

func ensureDownloadTokensCollection(app *pocketbase.PocketBase) error {
	_, err := app.FindCollectionByNameOrId("download_tokens")
	if err == nil {
		return nil
	}
	collection := core.NewBaseCollection("download_tokens")
	collection.Fields.Add(
		&core.TextField{
			Name:     "token",
			Required: true,
		},
		&core.DateField{
			Name:     "expires",
			Required: true,
		},
		&core.RelationField{
			Name:          "user",
			Required:      true,
			CollectionId:  "users",
			MaxSelect:     1,
			CascadeDelete: true,
		},
	)
	collection.Indexes = []string{
		"CREATE UNIQUE INDEX idx_token ON download_tokens (token)",
	}
	return app.Save(collection)
}
