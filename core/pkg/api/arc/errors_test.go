// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/arc"
	"github.com/synnaxlabs/x/errors"
)

var _ = Describe("ErrorProvider", func() {
	codec := errors.NewCodec(arc.ErrorProvider())
	It("Should round-trip a compile error", func() {
		err := arc.CompileError{Diagnostics: "1:1 unexpected token"}
		pld := codec.Encode(errors.Wrap(err, "compile"), false)
		Expect(pld).To(Equal(errors.Payload{
			Type: "arc",
			Data: "1:1 unexpected token",
		}))
		Expect(codec.Decode(pld)).To(Equal(err))
	})
	It("Should leave other errors to other providers", func() {
		Expect(codec.Encode(errors.New("cat"), false).Type).
			To(Equal(errors.TypeUnknown))
	})
})
