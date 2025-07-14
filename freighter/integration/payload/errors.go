// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package payload

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/synnaxlabs/x/errors"
)

type Error struct {
	Code    int
	Message string
}

var _ error = Error{}

func (e Error) Error() string { return e.Message }

func encodeError(_ context.Context, err error) (errors.Payload, bool) {
	var e Error
	ok := errors.As(err, &e)
	if !ok {
		return errors.Payload{}, false
	}
	return errors.Payload{
		Type: "integration.error",
		Data: fmt.Sprintf("%d,%s", e.Code, e.Message),
	}, true
}

func decodeError(_ context.Context, pld errors.Payload) (error, bool) {
	if pld.Type != "integration.error" {
		return nil, false
	}
	parts := strings.Split(pld.Data, ",")
	if len(parts) != 2 {
		return errors.New("unexpected error format"), true
	}
	code, err := strconv.Atoi(parts[0])
	if err != nil {
		return err, true
	}
	return Error{Code: code, Message: parts[1]}, true
}

func init() { errors.Register(encodeError, decodeError) }
