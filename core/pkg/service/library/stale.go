// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package library

import (
	"context"

	"github.com/synnaxlabs/x/errors"
)

// ErrStale is returned when a save sends the hash of a library document that changed
// after the writer read it.
var ErrStale = errors.New("changed since it was read")

// CheckFresh returns ErrStale when read, the hash a writer read a document at, differs
// from stored, the hash of the stored document.
func CheckFresh(read, stored string) error {
	if read != stored {
		return ErrStale
	}
	return nil
}

const staleType = "sy.library.stale"

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
