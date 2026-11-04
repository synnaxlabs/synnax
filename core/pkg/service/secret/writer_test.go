// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
	var (
		e  *env
		rk ontology.ID
	)
	BeforeEach(func(ctx SpecContext) {
		e = newEnv(ctx)
		rk = rack.Key(1).OntologyID()
	})

	resolve := func(ctx SpecContext, key secret.Key) string {
		GinkgoHelper()
		grant := MustSucceed(e.svc.Grant(ctx, nil, rk, key))
		return string(MustSucceed(e.svc.Resolve(ctx, nil, rk, grant, key))[key])
	}

	Describe("Create", func() {
		It(
			"Should give a stored secret a key and encrypt its value",
			func(ctx SpecContext) {
				s := e.createStored(ctx, "Password", "hunter2")
				Expect(s.Base().Key).ToNot(Equal(uuid.Nil()))
				Expect(s.Base().Updated).To(Equal(e.now))
				Expect(resolve(ctx, s.Base().Key)).To(Equal("hunter2"))
			},
		)
		It("Should keep a key the caller gives", func(ctx SpecContext) {
			key := uuid.New()
			s := secret.Secret{Variant: secret.StoredSecret{
				Base: secret.Base{Key: key, Name: "Password"},
			}}
			Expect(e.svc.NewWriter(nil).Create(ctx, &s, []byte("a"))).To(Succeed())
			Expect(e.retrieve(ctx, key).Base().Name).To(Equal("Password"))
		})
		It("Should save the path of a file secret", func(ctx SpecContext) {
			s := secret.Secret{Variant: secret.FileSecret{
				Base: secret.Base{Name: "Client key"},
				Path: "/etc/plant/key.pem",
			}}
			Expect(e.svc.NewWriter(nil).Create(ctx, &s, nil)).To(Succeed())
			file := e.retrieve(ctx, s.Base().Key).Variant.(secret.FileSecret)
			Expect(file.Path).To(Equal("/etc/plant/key.pem"))
		})
		It(
			"Should replace the value of the secret with the same key",
			func(ctx SpecContext) {
				s := e.createStored(ctx, "Password", "old")
				e.now += 10
				Expect(
					e.svc.NewWriter(nil).Create(ctx, &s, []byte("new")),
				).To(Succeed())
				Expect(e.retrieve(ctx, s.Base().Key).Base().Updated).To(Equal(e.now))
				Expect(resolve(ctx, s.Base().Key)).To(Equal("new"))
			},
		)
		DescribeTable("Should reject an invalid secret",
			func(ctx SpecContext, s secret.Secret, value, msg string) {
				Expect(e.svc.NewWriter(nil).Create(ctx, &s, []byte(value))).
					To(MatchError(ContainSubstring(msg)))
			},
			Entry(
				"a stored secret with no value",
				secret.Secret{
					Variant: secret.StoredSecret{Base: secret.Base{Name: "A"}},
				},
				"",
				"value: required",
			),
			Entry(
				"a file secret with no path",
				secret.Secret{Variant: secret.FileSecret{Base: secret.Base{Name: "A"}}},
				"",
				"path: required",
			),
			Entry(
				"a file secret with a value",
				secret.Secret{Variant: secret.FileSecret{
					Base: secret.Base{Name: "A"},
					Path: "/a",
				}},
				"v",
				"value: must be empty for a file secret",
			),
			Entry(
				"a secret with no name",
				secret.Secret{Variant: secret.StoredSecret{}},
				"v",
				"name: required",
			),
			Entry("a secret with no source", secret.Secret{}, "v", "source: required"),
		)
	})

	Describe("Rename", func() {
		It("Should rename the secret and keep its value", func(ctx SpecContext) {
			s := e.createStored(ctx, "Before", "hunter2")
			Expect(
				e.svc.NewWriter(nil).Rename(ctx, s.Base().Key, "After"),
			).To(Succeed())
			Expect(e.retrieve(ctx, s.Base().Key).Base().Name).To(Equal("After"))
			Expect(resolve(ctx, s.Base().Key)).To(Equal("hunter2"))
		})
		It("Should reject an empty name", func(ctx SpecContext) {
			s := e.createStored(ctx, "Before", "hunter2")
			Expect(e.svc.NewWriter(nil).Rename(ctx, s.Base().Key, "")).
				To(MatchError(ContainSubstring("name: required")))
		})
		It("Should return not found for a missing secret", func(ctx SpecContext) {
			Expect(e.svc.NewWriter(nil).Rename(ctx, uuid.New(), "A")).
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Delete", func() {
		It("Should delete the secret and its resource", func(ctx SpecContext) {
			s := e.createStored(ctx, "Password", "hunter2")
			Expect(e.svc.NewWriter(nil).Delete(ctx, s.Base().Key)).To(Succeed())
			Expect(e.svc.NewRetrieve().
				Where(secret.MatchKeys(s.Base().Key)).
				Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
			Expect(e.otg.NewRetrieve().
				WhereIDs(s.OntologyID()).
				Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
		})
		It("Should succeed for a missing secret", func(ctx SpecContext) {
			Expect(e.svc.NewWriter(nil).Delete(ctx, uuid.New())).To(Succeed())
		})
		It(
			"Should refuse to delete a secret that something uses",
			func(ctx SpecContext) {
				s := e.createStored(ctx, "Plant key", "k")
				e.uses = []secret.Use{
					{
						Secret: s.Base().Key,
						Resource: ontology.ID{
							Type: ontology.ResourceTypeCertificate,
							Key:  "c",
						},
						Name: "Plant certificate",
					},
				}
				Expect(e.svc.NewWriter(nil).Delete(ctx, s.Base().Key)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(
						`cannot delete secret "Plant key" because certificate `+
							`"Plant certificate" uses it`,
					)),
				))
				Expect(e.retrieve(ctx, s.Base().Key).Base().Name).To(Equal("Plant key"))
			},
		)
	})
})
