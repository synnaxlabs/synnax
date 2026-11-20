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
	"io/fs"
	"testing/fstest"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
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
	granted := func(subject resource.ID, keys ...secret.Key) grant.Grant {
		return grant.Grant{Subject: subject, Command: cmd, Secrets: keys}
	}
	BeforeEach(func(ctx SpecContext) {
		e = newStore(ctx)
		e.folder = fstest.MapFS{
			"k.pem":      {Data: []byte("client key")},
			"lf.txt":     {Data: []byte("token\n")},
			"crlf.txt":   {Data: []byte("token\r\n")},
			"lines.txt":  {Data: []byte("token\n\r\n\n")},
			"spaces.txt": {Data: []byte(" a token  \r\n")},
		}
		e.svc = MustOpen(openService(ctx, e, secret.NewEncryptionKey()))
		rk = coreRack.ResourceID()
		rk2 = otherRack.ResourceID()
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
				g := MustSucceed(e.svc.Grant(ctx, e.db, granted(rk, key)))
				Expect(storedGrants(ctx, e)).To(ConsistOf(SatisfyAll(
					HaveField("Key", g),
					HaveField("Subject", rk),
					HaveField("Command", cmd),
					HaveField("Secrets", []secret.Key{key}),
				)))
			},
		)
		It("Should return not found for a missing secret", func(ctx SpecContext) {
			Expect(e.svc.Grant(ctx, e.db, granted(rk, key, uuid.New()))).Error().
				To(MatchError(query.ErrNotFound))
			Expect(storedGrants(ctx, e)).To(BeEmpty())
		})
		It("Should store a file secret of the subject rack", func(ctx SpecContext) {
			s := e.createFile(ctx, "Client key", "k.pem", otherRack)
			MustSucceed(e.svc.Grant(ctx, e.db, granted(rk2, s.Base().Key)))
			Expect(storedGrants(ctx, e)).To(ConsistOf(
				HaveField("Secrets", []secret.Key{s.Base().Key}),
			))
		})
		It("Should refuse a file secret of another rack", func(ctx SpecContext) {
			s := e.createFile(ctx, "Client key", "k.pem", otherRack)
			Expect(
				e.svc.Grant(ctx, e.db, granted(rk, s.Base().Key)),
			).Error().
				To(MatchError(
					`secret "Client key" is a file of rack 2, not of ` + rk.String() +
						": validation error",
				))
			Expect(storedGrants(ctx, e)).To(BeEmpty())
		})
		It("Should reject a grant for a subject that is not a rack",
			func(ctx SpecContext) {
				user := resource.ID{
					Type: resource.TypeUser,
					Key:  uuid.New().String(),
				}
				Expect(e.svc.Grant(ctx, e.db, granted(user, key))).Error().
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
				g := MustSucceed(e.svc.Grant(ctx, e.db, granted(rk, key)))
				spent, values := MustSucceed2(e.svc.Resolve(ctx, rk, cmd.Task, g))
				Expect(spent.Command).To(Equal(cmd))
				Expect(spent.Secrets).To(Equal([]secret.Key{key}))
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
			g := MustSucceed(e.svc.Grant(ctx, e.db, granted(rk2, key)))
			Expect(e.svc.Resolve(ctx, rk, cmd.Task, g)).Error().To(SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(ContainSubstring("grant is not for rack:1")),
			))
		})
		It("Should deny and spend a grant for another task", func(ctx SpecContext) {
			g := MustSucceed(e.svc.Grant(ctx, e.db, granted(rk, key)))
			other := uuid.New()
			Expect(e.svc.Resolve(ctx, rk, other, g)).Error().To(SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(ContainSubstring("grant is not for task "+other.String())),
			))
			Expect(storedGrants(ctx, e)).To(BeEmpty())
		})
		It("Should return not found for a secret deleted after the grant",
			func(ctx SpecContext) {
				g := MustSucceed(e.svc.Grant(ctx, e.db, granted(rk, key)))
				Expect(gorp.NewDelete[secret.Key, secret.Secret]().
					Where(gorp.MatchKeys[secret.Key, secret.Secret](key)).
					Exec(ctx, e.db)).To(Succeed())
				Expect(e.svc.Resolve(ctx, rk, cmd.Task, g)).Error().
					To(MatchError(query.ErrNotFound))
			},
		)
		It("Should read a file secret of the Core's rack from its secrets folder",
			func(ctx SpecContext) {
				s := e.createFile(ctx, "Client key", "k.pem", coreRack)
				g := MustSucceed(e.svc.Grant(ctx, e.db, granted(rk, s.Base().Key)))
				_, values := MustSucceed2(e.svc.Resolve(ctx, rk, cmd.Task, g))
				Expect(values).To(Equal(
					map[secret.Key][]byte{s.Base().Key: []byte("client key")},
				))
			},
		)
		It("Should name a file secret of the reading rack with no value", func(
			ctx SpecContext,
		) {
			s := e.createFile(ctx, "Client key", "k.pem", otherRack)
			g := MustSucceed(e.svc.Grant(ctx, e.db, granted(rk2, key, s.Base().Key)))
			spent, values := MustSucceed2(e.svc.Resolve(ctx, rk2, cmd.Task, g))
			Expect(spent.Secrets).To(Equal([]secret.Key{key, s.Base().Key}))
			Expect(values).To(Equal(map[secret.Key][]byte{key: []byte("hunter2")}))
		})
		It("Should not decrypt a value moved to another secret", func(ctx SpecContext) {
			other := e.createStored(ctx, "Token", "t")
			moved := password
			moved.SetBase(other.Base())
			Expect(gorp.NewCreate[secret.Key, secret.Secret]().
				Entry(&moved).
				Exec(ctx, e.db)).To(Succeed())
			g := MustSucceed(e.svc.Grant(ctx, e.db, granted(rk, other.Base().Key)))
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
		It("Should read a file secret of the Core's rack from its secrets folder",
			func(ctx SpecContext) {
				s := e.createFile(ctx, "Client key", "k.pem", coreRack)
				Expect(e.svc.Reveal(ctx, e.db, s.Base().Key)).To(Equal(
					map[secret.Key][]byte{s.Base().Key: []byte("client key")},
				))
			},
		)
		DescribeTable(
			"Should drop the line endings at the end of a file secret",
			func(ctx SpecContext, fileName, value string) {
				s := e.createFile(ctx, "Token", fileName, coreRack)
				Expect(e.svc.Reveal(ctx, e.db, s.Base().Key)).To(Equal(
					map[secret.Key][]byte{s.Base().Key: []byte(value)},
				))
			},
			Entry("a line feed", "lf.txt", "token"),
			Entry("a carriage return and a line feed", "crlf.txt", "token"),
			Entry("several line endings", "lines.txt", "token"),
			Entry(
				"spaces inside, at the start, and at the end",
				"spaces.txt",
				" a token  ",
			),
		)
		It("Should fail for a file that the secrets folder does not hold", func(
			ctx SpecContext,
		) {
			s := e.createFile(ctx, "Missing key", "missing.pem", coreRack)
			Expect(e.svc.Reveal(ctx, e.db, s.Base().Key)).Error().To(SatisfyAll(
				MatchError(fs.ErrNotExist),
				MatchError(ContainSubstring(`read secret "Missing key"`)),
			))
		})
		It("Should refuse a file secret when the Core has no secrets folder", func(
			ctx SpecContext,
		) {
			e := newEnv(ctx)
			s := e.createFile(ctx, "Client key", "k.pem", coreRack)
			Expect(e.svc.Reveal(ctx, e.db, s.Base().Key)).Error().To(MatchError(
				`secret "Client key" is a file, but the Core has no secrets folder: ` +
					"validation error",
			))
		})
		It("Should refuse a file secret of another rack", func(ctx SpecContext) {
			s := e.createFile(ctx, "Client key", "k.pem", otherRack)
			Expect(e.svc.Reveal(ctx, e.db, s.Base().Key)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(
					ContainSubstring(`secret "Client key" is a file, so rack 2`),
				),
			))
		})
	})
	Describe("RevealTo", func() {
		It("Should read a stored secret for any reader", func(ctx SpecContext) {
			Expect(e.svc.RevealTo(ctx, e.db, rk2, key)).
				To(Equal(map[secret.Key][]byte{key: []byte("hunter2")}))
		})
		It("Should read a file secret for the Core's rack", func(ctx SpecContext) {
			s := e.createFile(ctx, "Client key", "k.pem", coreRack)
			Expect(e.svc.RevealTo(ctx, e.db, rk, s.Base().Key)).To(Equal(
				map[secret.Key][]byte{s.Base().Key: []byte("client key")},
			))
		})
		It("Should return no value for a file secret of a rack that is not the Core's",
			func(ctx SpecContext) {
				s := e.createFile(ctx, "Client key", "k.pem", otherRack)
				Expect(e.svc.RevealTo(ctx, e.db, rk2, s.Base().Key)).To(BeEmpty())
			},
		)
		It("Should refuse a file secret of another rack", func(ctx SpecContext) {
			s := e.createFile(ctx, "Client key", "k.pem", coreRack)
			Expect(e.svc.RevealTo(ctx, e.db, rk2, s.Base().Key)).Error().To(MatchError(
				`secret "Client key" is a file, so rack 1 reads it from its own ` +
					"secrets folder: validation error",
			))
		})
		It("Should return not found for a missing secret", func(ctx SpecContext) {
			Expect(e.svc.RevealTo(ctx, e.db, rk, uuid.New())).Error().
				To(MatchError(query.ErrNotFound))
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
