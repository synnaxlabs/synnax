// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rbac

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/access"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

type Enforcer struct {
	DefaultEffect access.Effect
	Legislator    *Legislator
}

var _ access.Enforcer = (*Enforcer)(nil)

// Enforce implements the access.Enforcer interface.
func (e *Enforcer) Enforce(ctx context.Context, req access.Request) error {
	policies, err := e.Legislator.Retrieve(ctx, req.Subject, req.Object)
	if err != nil {
		if errors.Is(err, query.NotFound) {
			return e.defaultErr()
		}
		return err
	}
	for _, p := range policies {
		if p.Matches(req) {
			if p.Effect == access.Deny {
				return access.Denied
			}
			return access.Granted
		}
	}
	return e.defaultErr()
}

func (e *Enforcer) defaultErr() error {
	if e.DefaultEffect == access.Deny {
		return access.Denied
	}
	return access.Granted
}
