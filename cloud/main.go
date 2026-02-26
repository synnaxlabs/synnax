// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package main

import (
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"github.com/synnaxlabs/cloud/downloads"
	"github.com/synnaxlabs/cloud/migrations"
)

func main() {
	app := pocketbase.New()

	migrations.Register(app)

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		downloads.RegisterRoutes(se)
		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
