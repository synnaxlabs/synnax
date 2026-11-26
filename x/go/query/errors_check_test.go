// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package query_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

var _ = Describe("Multiple results errors (check)", func() {
	It("Should keep the text of a multiple results error across two hops", func() {
		codec := errors.NewCodec(query.ErrorProvider())
		err := query.MultipleResults(
			errors.New("cause"),
			"%d channels are named %q",
			2,
			"pt",
		)
		once := codec.Decode(codec.Encode(err, false))
		Expect(once).To(MatchError(err.Error()))
		twice := codec.Decode(codec.Encode(once, false))
		Expect(twice).To(MatchError(err.Error()))
		Expect(codec.Encode(twice, false).Data).To(Equal(err.Error()))
	})
})
