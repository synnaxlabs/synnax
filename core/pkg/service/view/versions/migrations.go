// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import (
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	v0 "github.com/synnaxlabs/synnax/pkg/service/view/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/view/versions/v1"
)

// NewMigrations returns the ordered migration chain for stored views. The ontology
// removes the resources of views the chain deletes.
func NewMigrations(otg *ontology.Ontology) []gorp.Migration {
	return []gorp.Migration{
		v0.NormalizeKeys,
		v0.Migration,
		v0.NameMigration,
		v1.NewMigration(otg),
	}
}
