// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package access

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/telem"
)

type Request struct {
	Subject ontology.ID
	Objects []ontology.ID
	Action  Action
	// TimeRange is the time range being accessed (for data read/write operations).
	// If a policy has a TimeRange constraint, the request's TimeRange must be within it.
	TimeRange telem.TimeRange
	// Properties is the list of properties being modified (for update operations).
	// If a policy has an AllowedProperties constraint, all properties must be in that list.
	Properties []string
}

type Enforcer interface {
	Enforce(context.Context, Request) error
}
