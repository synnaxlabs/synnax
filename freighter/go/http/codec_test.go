// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter/http"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Codec", func() {
	Describe("ResolveCodec", func() {
		It("Should determine the encoder based on a content-type", func() {
			codec := MustSucceed(http.ResolveCodec("application/json"))
			Expect(codec.ContentType()).To(Equal("application/json"))
		})
		It("Should return an error if the content-type is not supported", func() {
			Expect(http.ResolveCodec("application/octet-stream")).
				Error().To(MatchError(ContainSubstring("unable to determine encoding type")))
		})
	})
	Describe("SupportedContentTypes", func() {
		It("Should return the supported content types", func() {
			contentTypes := http.SupportedContentTypes()
			Expect(contentTypes).To(ContainElement("application/json"))
			Expect(contentTypes).To(ContainElement("application/msgpack"))
		})
	})
})
