// Copyright 2025 Synnax Labs, Inc.
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
)

type Request struct {
	Subject ontology.ID
	Objects []ontology.ID
	Action  Action
}

type Enforcer interface {
	Enforce(context.Context, Request) error
}
