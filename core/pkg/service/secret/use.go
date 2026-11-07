// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
)

// Use is a resource that uses a secret.
type Use struct {
	// Resource identifies the resource that uses the secret.
	Resource ontology.ID
	// Name is the name of the resource.
	Name string
	// Secret is the key of the secret in use.
	Secret Key
}

// Finder returns the uses of the secrets with the given keys, read through tx.
type Finder func(ctx context.Context, tx gorp.Tx, keys []Key) ([]Use, error)
