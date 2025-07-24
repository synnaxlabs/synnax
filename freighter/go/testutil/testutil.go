// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"context"

	"github.com/synnaxlabs/x/errors"
)

type (
	Request = struct {
		ID      int
		Message string
	}
	Response = Request
)

const testErrorType = "freighter.test"

var errTest = errors.New(testErrorType)

func init() {
	errors.Register(
		func(_ context.Context, err error) (errors.Payload, bool) {
			if errors.Is(err, errTest) {
				return errors.Payload{Type: testErrorType, Data: err.Error()}, true
			}
			return errors.Payload{}, false
		},
		func(context.Context, errors.Payload) (error, bool) { return nil, false },
	)
}
