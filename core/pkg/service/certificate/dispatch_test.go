// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package certificate_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Dispatch", func() {
	type frame = actions.Scoped[certificate.Key, certificate.Action]
	var (
		e *env
		c certificate.Certificate
	)
	write := func(ctx context.Context, fn func(certificate.Writer) error) error {
		return e.write(ctx, func(w certificate.Writer) error {
			return fn(w.WithDispatchKey("dk"))
		})
	}
	set := func(path string, value any) certificate.Action {
		return certificate.NewSetAction(
			certificate.SetPayload{Path: []string{path}, Value: value},
		)
	}
	apply := func(ctx context.Context, acts ...certificate.Action) error {
		return write(ctx, func(w certificate.Writer) error {
			return w.Apply(ctx, c.Base().Key, acts...)
		})
	}
	record := func() *Recorder[certificate.Key, certificate.Action] {
		rec := &Recorder[certificate.Key, certificate.Action]{}
		DeferCleanup(e.svc.Actions().OnChange(rec.Record))
		return rec
	}
	BeforeEach(func(ctx SpecContext) {
		e = newEnv(ctx, telem.Hour)
		key := newKey()
		c = e.create(
			ctx,
			stored("Before", newCertificate(key, "a", time.Hour)),
			encodeKey(key),
		)
	})

	It("Should store the edited name and rename the status", func(ctx SpecContext) {
		Expect(apply(ctx, set("name", "After"))).To(Succeed())
		got := e.retrieve(ctx, c.Base().Key)
		Expect(got.Base().Name).To(Equal("After"))
		Expect(got.Variant.(certificate.StoredCertificate).Pem).
			To(Equal(c.Variant.(certificate.StoredCertificate).Pem))
		Expect(e.expiry(ctx, c.Base().Key).Name).To(Equal("After"))
	})

	DescribeTable("Should refuse an edit", func(
		ctx SpecContext,
		act certificate.Action,
		matcher string,
	) {
		err := apply(ctx, act)
		Expect(err).To(MatchError(validate.ErrValidation))
		Expect(err).To(MatchError(ContainSubstring(matcher)))
		Expect(e.retrieve(ctx, c.Base().Key)).To(Equal(c))
	},
		Entry("of the chain", set("pem", "x"), `unknown object member name "pem"`),
		Entry(
			"of the private key",
			set("private_key", nil),
			`unknown object member name "private_key"`,
		),
		Entry(
			"of the source",
			set("source", "file"),
			`unknown object member name "source"`,
		),
		Entry(
			"that creates the certificate",
			certificate.NewCreateAction(certificate.CreatePayload{}),
			"a certificate create",
		),
	)

	It("Should refuse an empty name", func(ctx SpecContext) {
		Expect(apply(ctx, set("name", ""))).
			To(MatchError(ContainSubstring("name: required")))
		Expect(e.retrieve(ctx, c.Base().Key)).To(Equal(c))
	})

	It("Should broadcast a create with the certificate", func(ctx SpecContext) {
		rec := record()
		created := certificate.Certificate{Variant: certificate.FileCertificate{
			Base:            certificate.Base{Name: "Created"},
			CertificatePath: "/etc/plant.pem",
		}}
		Expect(write(ctx, func(w certificate.Writer) error {
			return w.Set(ctx, &created, nil)
		})).To(Succeed())
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(1))
		Expect(frames[0].DispatchKey).To(Equal("dk"))
		Expect(frames[0].Actions).To(Equal([]certificate.Action{
			certificate.NewCreateAction(
				certificate.CreatePayload{Certificate: created},
			),
		}))
	})

	It(
		"Should broadcast an applied batch with its dispatch key",
		func(ctx SpecContext) {
			rec := record()
			Expect(apply(ctx, set("name", "After"))).To(Succeed())
			frames := rec.Snapshot()
			Expect(frames).To(Equal([]frame{{
				Key:         c.Base().Key,
				DispatchKey: "dk",
				Seq:         frames[0].Seq,
				Actions:     []certificate.Action{set("name", "After")},
			}}))
		},
	)

	It("Should broadcast the set that a rename applies", func(ctx SpecContext) {
		rec := record()
		Expect(e.write(ctx, func(w certificate.Writer) error {
			return w.Rename(ctx, c.Base().Key, "After")
		})).To(Succeed())
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(1))
		Expect(frames[0].Actions).To(Equal([]certificate.Action{set("name", "After")}))
	})
})
