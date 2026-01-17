// Copyright 2026 Synnax Labs, Inc.
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
	"strconv"
	"strings"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/errors"
)

type Message struct {
	Message string `json:"message" msgpack:"message"`
	ID      int    `json:"id" msgpack:"id"`
}

type (
	Server       = freighter.StreamServer[Message, Message]
	Client       = freighter.StreamClient[Message, Message]
	ServerStream = freighter.ServerStream[Message, Message]
	ClientStream = freighter.ClientStream[Message, Message]
)

type TestError struct {
	Message string
	Code    int
}

func (t TestError) Error() string {
	return t.Message
}

func encodeTestError(_ context.Context, err error) (errors.Payload, bool) {
	var errTest TestError
	ok := errors.As(err, &errTest)
	if !ok {
		return errors.Payload{}, false
	}
	return errors.Payload{
		Type: "integration.error",
		Data: strconv.Itoa(errTest.Code) + "," + errTest.Message,
	}, true
}

func decodeTestError(_ context.Context, pld errors.Payload) (error, bool) {
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
	errors.Register(encodeTestError, decodeTestError)
}
