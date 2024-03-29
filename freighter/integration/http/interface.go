// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/ferrors"
	"strconv"
	"strings"
)

type Message struct {
	ID      int    `json:"id" msgpack:"id"`
	Message string `json:"message" msgpack:"message"`
}

type (
	Server       = freighter.StreamServer[Message, Message]
	Client       = freighter.StreamClient[Message, Message]
	ServerStream = freighter.ServerStream[Message, Message]
	ClientStream = freighter.ClientStream[Message, Message]
)

type TestError struct {
	Code    int
	Message string
}

func (t TestError) Error() string {
	return t.Message
}

func (t TestError) FreighterType() ferrors.Type {
	return "integration.error"
}

func encodeTestError(_ context.Context, err error) (ferrors.Payload, bool) {
	var te TestError
	ok := errors.As(err, &te)
	if !ok {
		return ferrors.Payload{}, false
	}
	return ferrors.Payload{
		Type: "integration.error",
		Data: strconv.Itoa(te.Code) + "," + te.Message,
	}, true
}

func decodeTestError(_ context.Context, pld ferrors.Payload) (error, bool) {
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
	return TestError{
		Code:    code,
		Message: parts[1],
	}, true
}

func init() {
	ferrors.Register(encodeTestError, decodeTestError)
}
