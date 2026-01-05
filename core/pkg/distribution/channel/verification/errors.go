// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package verification

import (
	"github.com/synnaxlabs/x/encoding/base64"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/types"
)

var (
	ErrInvalid = errors.New(base64.MustDecode("aW52YWxpZCBsaWNlbnNlIGtleQo="))
	ErrFree    = errors.Newf(
		base64.MustDecode(
			"dXNpbmcgbW9yZSB0aGFuICVkIGNoYW5uZWxzIHdpdGhvdXQgYSBsaWNlbnNlIGtleQ==",
		),
		FreeCount,
	)

	ErrTooMany = errors.New(base64.MustDecode(
		"dXNpbmcgbW9yZSBjaGFubmVscyB0aGFuIGFsbG93ZWQgYnkgdGhlIGxpY2Vuc2Uga2V5",
	))
	ErrStale = errors.Newf(
		base64.MustDecode(
			"dXNpbmcgbW9yZSB0aGFuICVkIGNoYW5uZWxzIHdpdGggYW4gZXhwaXJlZCBsaWNlbnNlIGtleQ==",
		),
		FreeCount,
	)
)

var errTooManyWrapString = base64.MustDecode("bGltaXQgaXMgJWQgY2hhbm5lbHM=")

func newErrTooMany(count types.Uint20) error {
	return errors.Wrapf(ErrTooMany, errTooManyWrapString, count)
}
