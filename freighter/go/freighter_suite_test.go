// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package freighter_test

import (
	"context"
	roacherrors "github.com/cockroachdb/errors"
	"github.com/synnaxlabs/freighter/ferrors"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type request struct {
	ID      int    `json:"id" msgpack:"id"`
	Message string `json:"message" msgpack:"message"`
}

type response struct {
	ID      int    `json:"id" msgpack:"id"`
	Message string `json:"message" msgpack:"message"`
}

var myCustomError = ferrors.Typed(roacherrors.New("my custom error"), "myCustomError")

func TestGo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Freighter Suite")
}

var _ = BeforeSuite(func() {
	ferrors.Register(
		func(_ context.Context, err error) (ferrors.Payload, bool) {
			v, ok := err.(ferrors.Error)
			if !ok || v.FreighterType() != "myCustomError" {
				return ferrors.Payload{}, false
			}
			return ferrors.Payload{
				Type: "myCustomError",
				Data: v.Error(),
			}, true
		},
		func(ctx context.Context, f ferrors.Payload) (error, bool) {
			if f.Type != "myCustomError" {
				return nil, false
			}
			return myCustomError, true
		},
	)
})
