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
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/secret/grant"
	"github.com/synnaxlabs/x/testutil"
)

// OpenService opens a secret service on db with a new encryption key, along with the
// grant service it needs. uses finds the uses of each secret. The services close when
// the spec or container that opens them ends.
func OpenService(
	ctx context.Context,
	db *gorp.DB,
	otg *ontology.Ontology,
	g *group.Service,
	uses secret.Finder,
) *secret.Service {
	ginkgo.GinkgoHelper()
	grants := testutil.MustOpen(grant.OpenService(ctx, grant.ServiceConfig{DB: db}))
	return testutil.MustOpen(secret.OpenService(ctx, secret.ServiceConfig{
		DB:            db,
		Ontology:      otg,
		Group:         g,
		Uses:          uses,
		Grant:         grants,
		EncryptionKey: secret.NewEncryptionKey(),
	}))
}
