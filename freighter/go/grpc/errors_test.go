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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/x/errors"
)

var _ = Describe("EncodeError", func() {
	It("Should encode a registered error into a payload that decodes back", func() {
		pld := grpc.EncodeError(errorCodec(), freighter.ErrStreamClosed, false)
		Expect(pld.Type).To(Equal("freighter.stream_closed"))
		Expect(errorCodec().Decode(errors.Payload{Type: pld.Type, Data: pld.Data})).
			To(MatchError(freighter.ErrStreamClosed))
	})

	It("Should encode a nil error as a nil payload", func() {
		Expect(
			grpc.EncodeError(errorCodec(), nil, false).Type,
		).To(Equal(errors.TypeNil))
	})

	It("Should encode an unregistered error with its message", func() {
		pld := grpc.EncodeError(errorCodec(), errors.New("cat"), false)
		Expect(pld.Type).To(Equal(errors.TypeUnknown))
		Expect(pld.Data).To(Equal("cat"))
	})
})
