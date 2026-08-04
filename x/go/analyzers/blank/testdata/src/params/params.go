// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package params

import "context"

func Check(_ *context.Context) error { return nil } // want `all parameters are blank; parameter names can be removed`

func Pair(_ context.Context, _ string) {} // want `all parameters are blank; parameter names can be removed`

func Shared(_, _ int) {} // want `all parameters are blank; parameter names can be removed`

func Variadic(_ string, _ ...int) {} // want `all parameters are blank; parameter names can be removed`

type handler struct{}

func (h handler) Handle(_ context.Context, _ string) error { // want `all parameters are blank; parameter names can be removed`
	_ = h
	return nil
}
