// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package errors_test

import (
	roacherrors "github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/synnaxlabs/x/query"
)

var _ = Describe("Errors", func() {
	DescribeTable("Name Errors", func(err error, expectedType errors.Type, expectedError error) {
		Expect(err.Type).To(Equal(expectedType))
		if expectedError == nil {
			Expect(err.Err).To(BeNil())
		} else {
			// Because these errors are exposed through an API, we test for comparison by Message.
			Expect(err.Error()).To(Equal(expectedError.Error()))
		}
	},
		Entry("General", errors.General(roacherrors.New("error")), errors.TypeGeneral, roacherrors.New("error")),
		Entry("Unexpected", errors.Unexpected(roacherrors.New("error")), errors.TypeUnexpected, roacherrors.New("error")),
		Entry("Parse", errors.Parse(roacherrors.New("error")), errors.TypeParse, roacherrors.New("error")),
		Entry("AuthService", errors.Auth(roacherrors.New("error")), errors.TypeAuth, roacherrors.New("error")),
		Entry("Params", errors.Query(query.NotFound), errors.TypeQuery, query.NotFound),
		Entry("Params", errors.Query(query.UniqueViolation), errors.TypeQuery, query.UniqueViolation),
	)
})
