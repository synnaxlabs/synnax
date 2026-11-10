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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/secret/grant"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Grants", func() {
	var (
		e        *env
		rk, rk2  resource.ID
		password secret.Secret
		key      secret.Key
	)
	BeforeEach(func(ctx SpecContext) {
		e = newEnv(ctx)
		rk = rack.Key(1).ResourceID()
		rk2 = rack.Key(2).ResourceID()
		password = e.createStored(ctx, "Password", "hunter2")
		key = password.Base().Key
	})

	Describe("Grant", func() {
		It("Should let the rack read the secrets it names once", func(ctx SpecContext) {
			g := MustSucceed(e.svc.Grant(ctx, e.db, rk, key))
			Expect(e.svc.Resolve(ctx, rk, g, key)).
				To(Equal(map[secret.Key][]byte{key: []byte("hunter2")}))
			Expect(e.svc.Resolve(ctx, rk, g, key)).Error().To(SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(ContainSubstring(
					"grant "+g.String()+" has ended or does not exist",
				)),
			))
		})
		It("Should store the subject and the secrets", func(ctx SpecContext) {
			g := MustSucceed(e.svc.Grant(ctx, e.db, rk, key))
			Expect(storedGrants(ctx, e)).To(ConsistOf(SatisfyAll(
				HaveField("Key", g),
				HaveField("Subject", rk),
				HaveField("Secrets", []secret.Key{key}),
			)))
		})
		It("Should return not found for a missing secret", func(ctx SpecContext) {
			Expect(e.svc.Grant(ctx, e.db, rk, key, uuid.New())).Error().
				To(MatchError(query.ErrNotFound))
			Expect(storedGrants(ctx, e)).To(BeEmpty())
		})
		It("Should reject a grant for a subject that is not a rack",
			func(ctx SpecContext) {
				user := resource.ID{
					Type: resource.TypeUser,
					Key:  uuid.New().String(),
				}
				Expect(e.svc.Grant(ctx, e.db, user, key)).Error().
					To(SatisfyAll(
						BeAssignableToTypeOf(validate.PathError{}),
						MatchError("subject: "+user.String()+" is not a rack"),
					))
			},
		)
	})

	Describe("Resolve", func() {
		It("Should deny a rack with no grant", func(ctx SpecContext) {
			Expect(e.svc.Resolve(ctx, rk, uuid.Nil(), key)).Error().To(SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(ContainSubstring("has ended or does not exist")),
			))
		})
		It("Should deny a secret the grant does not name", func(ctx SpecContext) {
			other := e.createStored(ctx, "Token", "t")
			g := MustSucceed(e.svc.Grant(ctx, e.db, rk, key))
			Expect(e.svc.Resolve(ctx, rk, g, other.Base().Key)).Error().
				To(SatisfyAll(
					MatchError(access.ErrDenied),
					MatchError(ContainSubstring(`rack:1 cannot read secret "Token"`)),
				))
		})
		It("Should deny a grant for another rack", func(ctx SpecContext) {
			g := MustSucceed(e.svc.Grant(ctx, e.db, rk2, key))
			Expect(e.svc.Resolve(ctx, rk, g, key)).Error().To(SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(ContainSubstring("grant is not for rack:1")),
			))
		})
		It("Should return not found for a missing secret", func(ctx SpecContext) {
			Expect(e.svc.Resolve(ctx, rk, uuid.Nil(), uuid.New())).Error().
				To(MatchError(query.ErrNotFound))
		})
		It("Should refuse to resolve a file secret", func(ctx SpecContext) {
			s := secret.Secret{Variant: secret.FileSecret{
				Base:     secret.Base{Name: "Client key"},
				FileName: "k.pem",
			}}
			Expect(e.write(ctx, func(w secret.Writer) error {
				return w.Set(ctx, &s, nil)
			})).To(Succeed())
			g := MustSucceed(e.svc.Grant(ctx, e.db, rk, s.Base().Key))
			Expect(e.svc.Resolve(ctx, rk, g, s.Base().Key)).Error().
				To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(`secret "Client key" is a file`)),
				))
		})
		It("Should not decrypt a value moved to another secret", func(ctx SpecContext) {
			other := e.createStored(ctx, "Token", "t")
			moved := password
			moved.SetBase(other.Base())
			Expect(gorp.NewCreate[secret.Key, secret.Secret]().
				Entry(&moved).
				Exec(ctx, e.db)).To(Succeed())
			g := MustSucceed(e.svc.Grant(ctx, e.db, rk, other.Base().Key))
			Expect(e.svc.Resolve(ctx, rk, g, other.Base().Key)).Error().
				To(MatchError(ContainSubstring(`decrypt secret "Token"`)))
		})
	})
})

// storedGrants returns every grant in the DB of e.
func storedGrants(ctx SpecContext, e *env) []grant.Grant {
	GinkgoHelper()
	var grants []grant.Grant
	Expect(gorp.NewRetrieve[grant.Key, grant.Grant]().
		Entries(&grants).
		Exec(ctx, e.db)).To(Succeed())
	return grants
}
