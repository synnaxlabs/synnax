// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil provides test helpers for working with racks.
package testutil

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
)

// CreateDriverRole defines a role in the ontology for a rack service to give new racks,
// and returns its key.
func CreateDriverRole(ctx context.Context, otg *ontology.Ontology) (role.Key, error) {
	k := uuid.New()
	if err := otg.NewWriter(nil).DefineResources(ctx, role.OntologyID(k)); err != nil {
		return uuid.Nil(), err
	}
	return k, nil
}
