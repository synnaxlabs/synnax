// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd

import (
	"context"

	"github.com/synnaxlabs/x/errors"
)

// ErrStale is returned when a save sends the hash of an ICD that changed after the
// writer read it.
var ErrStale = errors.New("changed since it was read")

const staleType = "sy.icd.stale"

func encode(_ context.Context, err error) (errors.Payload, bool) {
	if errors.CheapIs(err, ErrStale) {
		return errors.Payload{Type: staleType, Data: err.Error()}, true
	}
	return errors.Payload{}, false
}

func decode(_ context.Context, p errors.Payload) (error, bool) {
	if p.Type != staleType {
		return nil, false
	}
	return errors.Wrap(ErrStale, p.Data), true
}

func init() { errors.Register(encode, decode) }
