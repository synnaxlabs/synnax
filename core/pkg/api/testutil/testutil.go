// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil provides helpers for api layer test suites.
package testutil

import (
	"context"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/service/user"
)

// AuthedCtx returns a freighter.Context derived from ctx with the given user installed
// as the request subject, so auth.GetSubject succeeds inside an api service.
func AuthedCtx(ctx context.Context, u user.User) freighter.Context {
	fctx := freighter.Context{Context: ctx, Params: freighter.Params{}}
	fctx.Set("Subject", u.OntologyID())
	return fctx
}
