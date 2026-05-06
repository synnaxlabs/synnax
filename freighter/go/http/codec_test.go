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
)

var _ = Describe("Codec", func() {
	It("Should have the correct content type", func() {
		Expect(http.JSONCodec.ContentType()).To(Equal("application/json"))
		Expect(http.MsgPackCodec.ContentType()).To(Equal("application/msgpack"))
	})
})
