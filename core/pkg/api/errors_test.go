// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api_test

import (
	"context"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/freighter/recovery"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/arc"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/license"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var _ = Describe("NewErrorCodec", func() {
	codec := api.NewErrorCodec()
	DescribeTable(
		"Should encode every error an API handler returns as a registered type",
		func(err error, errType string) {
			pld := codec.Encode(err, false)
			Expect(pld.Type).To(Equal(errType))
			Expect(codec.Decode(pld)).To(MatchError(err))
		},
		Entry("EOF", io.EOF, "freighter.eof"),
		Entry("stream closed", freighter.ErrStreamClosed, "freighter.stream_closed"),
		Entry("panic", recovery.ErrPanic, "sy.internal"),
		Entry("not found", query.ErrNotFound, "sy.query.not_found"),
		Entry(
			"unique violation",
			query.ErrUniqueViolation,
			"sy.query.unique_violation",
		),
		Entry(
			"invalid parameters",
			query.ErrInvalidParameters,
			"sy.query.invalid_parameters",
		),
		Entry("unsupported", query.ErrUnsupported, "sy.query.unsupported"),
		Entry("validation", validate.ErrValidation, "sy.validation"),
		Entry("unauthorized", control.ErrUnauthorized, "sy.control.unauthorized"),
		Entry(
			"invalid credentials",
			auth.ErrInvalidCredentials,
			"sy.auth.invalid-credentials",
		),
		Entry(
			"missing credentials",
			auth.ErrMissingCredentials,
			"sy.auth.missing_credentials",
		),
		Entry(
			"repeated username",
			auth.ErrRepeatedUsername,
			"sy.auth.repeated-username",
		),
		Entry("invalid token", auth.ErrInvalidToken, "sy.auth.invalid_token"),
		Entry("expired token", auth.ErrExpiredToken, "sy.auth.expired_token"),
		Entry("session ended", auth.ErrSessionEnded, "sy.auth.session_ended"),
		Entry("access denied", auth.ErrAccessDenied, "sy.auth.access_denied"),
		Entry("missing license", license.ErrMissing, "sy.license.missing"),
		Entry("expired license", license.ErrExpired, "sy.license.expired"),
		Entry("invalid license", license.ErrInvalid, "sy.license.invalid"),
		Entry("license edition", license.ErrEdition, "sy.license.edition"),
		Entry("license fingerprint", license.ErrFingerprint, "sy.license.fingerprint"),
		Entry("too many channels", license.ErrTooMany, "sy.license.too_many"),
	)
	It("Should encode an Arc compile error", func() {
		Expect(codec.Encode(arc.CompileError{}, false).Type).To(Equal("arc"))
	})
	It("Should encode a panic outside the API middleware as an internal error", func() {
		interceptor := fgrpc.RecoveryUnaryServerInterceptor(alamos.Instrumentation{})
		Expect(interceptor(
			context.Background(),
			nil,
			&grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"},
			func(context.Context, any) (any, error) { panic("boom") },
		)).Error().To(WithTransform(
			func(err error) errors.Payload {
				return errors.ParsePayload(status.Convert(err).Message())
			},
			SatisfyAll(
				HaveField("Type", "sy.internal"),
				WithTransform(codec.Decode, MatchError(recovery.ErrPanic)),
			),
		))
	})
	It("Should report a provider's errors as expected", func() {
		Expect(codec.Expected(io.EOF)).To(BeTrue())
		Expect(codec.Expected(errors.New("cat"))).To(BeFalse())
	})
})
