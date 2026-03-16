// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package httputil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/httputil"
)

var _ = Describe("Codec", func() {
	Describe("Determine", func() {
		It("Should determine the encoder based on a content-type", func() {
			ecd, err := httputil.ResolveCodec("application/json")
			Expect(err).ToNot(HaveOccurred())
			Expect(ecd.ContentType()).To(Equal("application/json"))
		})
		It("Should return an error if the content-type is not supported", func() {
			_, err := httputil.ResolveCodec("application/octet-stream")
			Expect(err).To(HaveOccurred())
		})
	})
	Describe("SupportedContentTypes", func() {
		It("Should return the supported content types", func() {
			contentTypes := httputil.SupportedContentTypes()
			Expect(contentTypes).To(ContainElement("application/json"))
			Expect(contentTypes).To(ContainElement("application/msgpack"))
		})
	})
})
