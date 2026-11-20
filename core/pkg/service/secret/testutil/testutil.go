// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil opens secret services for tests.
package testutil

import (
	"context"

	"github.com/onsi/ginkgo/v2"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/secret/grant"
	"github.com/synnaxlabs/x/testutil"
)

// CoreRack is the Core's rack of a service that OpenService opens, unless its configs
// set another.
const CoreRack rack.Key = 1

// OpenService opens a secret service on db with a new encryption key, along with the
// grant service it needs. cfgs override the defaults, such as the Core's rack and the
// secrets folder. The services close when the spec or container that opens them ends.
func OpenService(
	ctx context.Context,
	db *gorp.DB,
	cfgs ...secret.ServiceConfig,
) *secret.Service {
	ginkgo.GinkgoHelper()
	grants := testutil.MustOpen(grant.OpenService(ctx, grant.ServiceConfig{DB: db}))
	return testutil.MustOpen(secret.OpenService(ctx, append([]secret.ServiceConfig{{
		DB:            db,
		Grant:         grants,
		EncryptionKey: secret.NewEncryptionKey(),
		Rack:          CoreRack,
	}}, cfgs...)...))
}
