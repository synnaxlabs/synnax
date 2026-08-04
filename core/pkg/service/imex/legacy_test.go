// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package imex_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

var _ = Describe("PeekVersion", func() {
	It("Should report version 0 for a nil blob", func() {
		Expect(imex.PeekVersion(nil, "test data")).To(Equal(imex.Version(0)))
	})

	It("Should report version 0 for a blob without a version field", func() {
		blob := msgpack.EncodedJSON{"name": "n"}
		Expect(imex.PeekVersion(blob, "test data")).To(Equal(imex.Version(0)))
	})

	DescribeTable("Should peek the stamped version",
		func(v any, expected imex.Version) {
			blob := msgpack.EncodedJSON{"version": v}
			Expect(imex.PeekVersion(blob, "test data")).To(Equal(expected))
		},
		Entry("numeric", 3, imex.Version(3)),
		Entry("legacy semver string", "2.0.0", imex.Version(2)),
	)

	It("Should error on a malformed version, naming the resource", func() {
		blob := msgpack.EncodedJSON{"version": "not-semver"}
		Expect(imex.PeekVersion(blob, "test data")).Error().To(SatisfyAll(
			MatchError(ContainSubstring("peek test data version")),
			MatchError(ContainSubstring("invalid version")),
		))
	})
})

var _ = Describe("DecodeBlob", func() {
	type payload struct {
		Name string `json:"name"`
	}

	It("Should decode a nil blob as a zero value", func() {
		Expect(imex.DecodeBlob[payload](nil, "test data", 0)).To(Equal(payload{}))
	})

	It("Should decode the blob as the target type", func() {
		blob := msgpack.EncodedJSON{"name": "n"}
		Expect(imex.DecodeBlob[payload](blob, "test data", 1)).
			To(Equal(payload{Name: "n"}))
	})

	It("Should error on a shape mismatch, naming the version and resource", func() {
		blob := msgpack.EncodedJSON{"name": 42}
		Expect(imex.DecodeBlob[payload](blob, "test data", 2)).Error().
			To(MatchError(ContainSubstring("decode v2 test data")))
	})
})
