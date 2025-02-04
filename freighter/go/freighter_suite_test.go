// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/errors"
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

var myCustomError = errors.New("my custom error")

func TestGo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Freighter Suite")
}

var _ = BeforeSuite(func() {
	errors.Register(
		func(_ context.Context, err error) (errors.Payload, bool) {
			if errors.Is(err, myCustomError) {
				return errors.Payload{
					Type: "myCustomError",
					Data: err.Error(),
				}, true
			}
			return errors.Payload{}, false
		},
		func(ctx context.Context, f errors.Payload) (error, bool) {
			if f.Type != "myCustomError" {
				return nil, false
			}
			return myCustomError, true
		},
	)
})
