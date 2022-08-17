package errors_test

import (
	"github.com/arya-analytics/delta/pkg/api/errors"
	"github.com/arya-analytics/x/query"
	roacherrors "github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Errors", func() {
	DescribeTable("New Errors", func(err errors.Typed, expectedType errors.Type, expectedError error) {
		Expect(err.Type).To(Equal(expectedType))
		if expectedError == nil {
			Expect(err.Err).To(BeNil())
		} else {
			// Because these errors are exposed through an API, we test for comparison by Message.
			Expect(err.Err.Error()).To(Equal(expectedError.Error()))
		}
	},
		Entry("General", errors.General(roacherrors.New("error")), errors.TypeGeneral, roacherrors.New("error")),
		Entry("MaybeGeneral Nil", errors.MaybeGeneral(nil), errors.TypeNil, nil),
		Entry("MaybeGeneral", errors.MaybeGeneral(roacherrors.New("error")), errors.TypeGeneral, roacherrors.New("error")),
		Entry("Unexpected", errors.Unexpected(roacherrors.New("error")), errors.TypeUnexpected, roacherrors.New("error")),
		Entry("MaybeUnexpected Nil", errors.MaybeUnexpected(nil), errors.TypeNil, nil),
		Entry("MaybeUnexpected", errors.MaybeUnexpected(roacherrors.New("error")), errors.TypeUnexpected, roacherrors.New("error")),
		Entry("Parse", errors.Parse(roacherrors.New("error")), errors.TypeParse, roacherrors.New("error")),
		Entry("MaybeParse Nil", errors.MaybeParse(nil), errors.TypeNil, nil),
		Entry("MaybeParse", errors.MaybeParse(roacherrors.New("error")), errors.TypeParse, roacherrors.New("error")),
		Entry("AuthService", errors.Auth(roacherrors.New("error")), errors.TypeAuth, roacherrors.New("error")),
		Entry("MaybeAuth Nil", errors.MaybeAuth(nil), errors.TypeNil, nil),
		Entry("MaybeAuth", errors.MaybeAuth(roacherrors.New("error")), errors.TypeAuth, roacherrors.New("error")),
		Entry("Query", errors.Query(query.NotFound), errors.TypeQuery, query.NotFound),
		Entry("Query", errors.Query(query.UniqueViolation), errors.TypeQuery, query.UniqueViolation),
		Entry("MaybeQuery Nil", errors.MaybeQuery(nil), errors.TypeNil, nil),
		Entry("MaybeQuery", errors.MaybeQuery(roacherrors.New("error")), errors.TypeGeneral, roacherrors.New("error")),
	)
})
