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
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
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
		cmd      task.Command
	)
	BeforeEach(func(ctx SpecContext) {
		e = newEnv(ctx)
		rk = rack.Key(1).ResourceID()
		rk2 = rack.Key(2).ResourceID()
		password = e.createStored(ctx, "Password", "hunter2")
		key = password.Base().Key
		cmd = task.Command{
			Task: uuid.New(),
			Type: "test_connection",
			Key:  "cmd",
			Args: msgpack.EncodedJSON{"host": "plc"},
		}
	})

	Describe("Grant", func() {
		It("Should store the subject, the command, and the secrets",
			func(ctx SpecContext) {
				g := MustSucceed(e.svc.Grant(ctx, e.db, rk, cmd, key))
				Expect(storedGrants(ctx, e)).To(ConsistOf(SatisfyAll(
					HaveField("Key", g),
					HaveField("Subject", rk),
					HaveField("Command", cmd),
					HaveField("Secrets", []secret.Key{key}),
				)))
			},
		)
		It("Should return not found for a missing secret", func(ctx SpecContext) {
			Expect(e.svc.Grant(ctx, e.db, rk, cmd, key, uuid.New())).Error().
				To(MatchError(query.ErrNotFound))
			Expect(storedGrants(ctx, e)).To(BeEmpty())
		})
		It("Should reject a grant for a subject that is not a rack",
			func(ctx SpecContext) {
				user := resource.ID{
					Type: resource.TypeUser,
					Key:  uuid.New().String(),
				}
				Expect(e.svc.Grant(ctx, e.db, user, cmd, key)).Error().
					To(SatisfyAll(
						BeAssignableToTypeOf(validate.PathError{}),
						MatchError("subject: "+user.String()+" is not a rack"),
					))
			},
		)
	})

	Describe("Resolve", func() {
		It("Should return the stored command and the values once",
			func(ctx SpecContext) {
				g := MustSucceed(e.svc.Grant(ctx, e.db, rk, cmd, key))
				stored, values := MustSucceed2(e.svc.Resolve(ctx, rk, cmd.Task, g))
				Expect(stored).To(Equal(cmd))
				Expect(values).To(Equal(map[secret.Key][]byte{key: []byte("hunter2")}))
				Expect(e.svc.Resolve(ctx, rk, cmd.Task, g)).Error().To(SatisfyAll(
					MatchError(access.ErrDenied),
					MatchError(ContainSubstring(
						"grant "+g.String()+" has ended or does not exist",
					)),
				))
			},
		)
		It("Should deny a rack with no grant", func(ctx SpecContext) {
			Expect(e.svc.Resolve(ctx, rk, cmd.Task, uuid.Nil())).Error().To(SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(ContainSubstring("has ended or does not exist")),
			))
		})
		It("Should deny a grant for another rack", func(ctx SpecContext) {
			g := MustSucceed(e.svc.Grant(ctx, e.db, rk2, cmd, key))
			Expect(e.svc.Resolve(ctx, rk, cmd.Task, g)).Error().To(SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(ContainSubstring("grant is not for rack:1")),
			))
		})
		It("Should deny and spend a grant for another task", func(ctx SpecContext) {
			g := MustSucceed(e.svc.Grant(ctx, e.db, rk, cmd, key))
			other := uuid.New()
			Expect(e.svc.Resolve(ctx, rk, other, g)).Error().To(SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(ContainSubstring("grant is not for task "+other.String())),
			))
			Expect(storedGrants(ctx, e)).To(BeEmpty())
		})
		It("Should return not found for a secret deleted after the grant",
			func(ctx SpecContext) {
				g := MustSucceed(e.svc.Grant(ctx, e.db, rk, cmd, key))
				Expect(gorp.NewDelete[secret.Key, secret.Secret]().
					Where(gorp.MatchKeys[secret.Key, secret.Secret](key)).
					Exec(ctx, e.db)).To(Succeed())
				Expect(e.svc.Resolve(ctx, rk, cmd.Task, g)).Error().
					To(MatchError(query.ErrNotFound))
			},
		)
		It("Should refuse to resolve a file secret", func(ctx SpecContext) {
			s := secret.Secret{Variant: secret.FileSecret{
				Base:     secret.Base{Name: "Client key"},
				FileName: "k.pem",
			}}
			Expect(e.write(ctx, func(w secret.Writer) error {
				return w.Set(ctx, &s, nil)
			})).To(Succeed())
			g := MustSucceed(e.svc.Grant(ctx, e.db, rk, cmd, s.Base().Key))
			Expect(e.svc.Resolve(ctx, rk, cmd.Task, g)).Error().
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
			g := MustSucceed(e.svc.Grant(ctx, e.db, rk, cmd, other.Base().Key))
			Expect(e.svc.Resolve(ctx, rk, cmd.Task, g)).Error().
				To(MatchError(ContainSubstring(`decrypt secret "Token"`)))
		})
	})

	Describe("Reveal", func() {
		It("Should read a secret that no rack uses or holds a grant for", func(
			ctx SpecContext,
		) {
			Expect(e.svc.Reveal(ctx, e.db, key)).
				To(Equal(map[secret.Key][]byte{key: []byte("hunter2")}))
		})
		It("Should return not found for a missing secret", func(ctx SpecContext) {
			Expect(e.svc.Reveal(ctx, e.db, uuid.New())).Error().
				To(MatchError(query.ErrNotFound))
		})
		It("Should refuse to reveal a file secret", func(ctx SpecContext) {
			s := secret.Secret{Variant: secret.FileSecret{
				Base:     secret.Base{Name: "Client key"},
				FileName: "k.pem",
			}}
			Expect(e.write(ctx, func(w secret.Writer) error {
				return w.Set(ctx, &s, nil)
			})).To(Succeed())
			Expect(e.svc.Reveal(ctx, e.db, s.Base().Key)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(`secret "Client key" is a file`)),
			))
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
