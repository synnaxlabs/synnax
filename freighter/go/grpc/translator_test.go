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

type requestTranslator struct{}

var _ grpc.Translator[testutil.Request, *v1.Request] = requestTranslator{}

func (requestTranslator) Forward(
	_ context.Context,
	req testutil.Request,
) (*v1.Request, error) {
	return &v1.Request{Id: int32(req.ID), Message: req.Message}, nil
}

func (requestTranslator) Backward(
	_ context.Context,
	req *v1.Request,
) (testutil.Request, error) {
	return testutil.Request{ID: int(req.Id), Message: req.Message}, nil
}

type responseTranslator struct{}

var _ grpc.Translator[testutil.Response, *v1.Response] = responseTranslator{}

func (responseTranslator) Forward(
	_ context.Context,
	res testutil.Response,
) (*v1.Response, error) {
	return &v1.Response{Id: int32(res.ID), Message: res.Message}, nil
}

func (responseTranslator) Backward(
	_ context.Context,
	res *v1.Response,
) (testutil.Response, error) {
	return testutil.Response{ID: int(res.Id), Message: res.Message}, nil
}
