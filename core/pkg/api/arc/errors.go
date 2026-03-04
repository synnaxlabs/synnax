// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"

	"github.com/synnaxlabs/x/errors"
)

const errorType = "arc"

// CompileError is returned when Arc source code fails to parse, analyze, or
// compile. The Diagnostics field contains the formatted diagnostic output.
type CompileError struct {
	Diagnostics string
}

func (e CompileError) Error() string { return e.Diagnostics }

func encode(_ context.Context, err error) (errors.Payload, bool) {
	var ce CompileError
	if errors.As(err, &ce) {
		return errors.Payload{Type: errorType, Data: ce.Diagnostics}, true
	}
	return errors.Payload{}, false
}

func decode(_ context.Context, p errors.Payload) (error, bool) {
	if p.Type != errorType {
		return nil, false
	}
	return CompileError{Diagnostics: p.Data}, true
}

func init() { errors.Register(encode, decode) }
