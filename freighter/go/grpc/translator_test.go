// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package grpc_test

import (
	"context"

	"github.com/synnaxlabs/freighter/grpc"
	v1 "github.com/synnaxlabs/freighter/grpc/v1"
	"github.com/synnaxlabs/freighter/testutil"
)

// The servers under test take the backward request and forward response translators
// alone, so a server that needs both directions again fails to compile.
type (
	requestForwardTranslator   struct{}
	requestBackwardTranslator  struct{}
	responseForwardTranslator  struct{}
	responseBackwardTranslator struct{}
	requestTranslator          struct {
		requestForwardTranslator
		requestBackwardTranslator
	}
	responseTranslator struct {
		responseForwardTranslator
		responseBackwardTranslator
	}
)

var (
	_ grpc.Translator[testutil.Request, *v1.Request]   = requestTranslator{}
	_ grpc.Translator[testutil.Response, *v1.Response] = responseTranslator{}
)

func (requestForwardTranslator) Forward(
	_ context.Context,
	req testutil.Request,
) (*v1.Request, error) {
	return &v1.Request{Id: int32(req.ID), Message: req.Message}, nil
}

func (requestBackwardTranslator) Backward(
	_ context.Context,
	req *v1.Request,
) (testutil.Request, error) {
	return testutil.Request{ID: int(req.Id), Message: req.Message}, nil
}

func (responseForwardTranslator) Forward(
	_ context.Context,
	res testutil.Response,
) (*v1.Response, error) {
	return &v1.Response{Id: int32(res.ID), Message: res.Message}, nil
}

func (responseBackwardTranslator) Backward(
	_ context.Context,
	res *v1.Response,
) (testutil.Response, error) {
	return testutil.Response{ID: int(res.Id), Message: res.Message}, nil
}
