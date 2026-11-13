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
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/secret/grant"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// env is a secret service on its own in-memory DB, with a clock that each spec
// controls.
type env struct {
	db     *gorp.DB
	grants *grant.Service
	svc    *secret.Service
	now    telem.TimeStamp
}

func openService(
	ctx context.Context,
	e *env,
	key secret.EncryptionKey,
) (*secret.Service, error) {
	GinkgoHelper()
	return secret.OpenService(ctx, secret.ServiceConfig{
		DB:            e.db,
		Grant:         e.grants,
		Now:           func() telem.TimeStamp { return e.now },
		EncryptionKey: key,
	})
}

// newStore returns an env with the services that a secret service needs, and no secret
// service.
func newStore(ctx SpecContext) *env {
	GinkgoHelper()
	e := &env{now: telem.SecondTS * 1000}
	e.db = DeferClose(gorp.Wrap(memkv.New()))
	e.grants = MustOpen(grant.OpenService(ctx, grant.ServiceConfig{DB: e.db}))
	return e
}

// device stands in for a record that uses a secret as its password.
type device struct {
	Key      string     `json:"key"      msgpack:"key"`
	Password secret.Key `json:"password" msgpack:"password"`
}

func (d device) GorpKey() string { return d.Key }

func (d device) EncodeOrc(w *orc.Writer) error {
	w.String(d.Key)
	w.Write(d.Password[:])
	return nil
}

func (d *device) DecodeOrc(r *orc.Reader) (err error) {
	if d.Key, err = r.String(); err != nil {
		return err
	}
	_, err = r.Read(d.Password[:])
	return err
}

func newEnv(ctx SpecContext) *env {
	GinkgoHelper()
	e := newStore(ctx)
	e.svc = MustOpen(openService(ctx, e, secret.NewEncryptionKey()))
	return e
}

func (e *env) createStored(ctx SpecContext, name, value string) secret.Secret {
	GinkgoHelper()
	s := secret.Secret{Variant: secret.StoredSecret{Base: secret.Base{Name: name}}}
	Expect(e.write(ctx, func(w secret.Writer) error {
		return w.Set(ctx, &s, []byte(value))
	})).To(Succeed())
	return s
}

// write runs fn with a writer on a new transaction.
func (e *env) write(ctx context.Context, fn func(secret.Writer) error) error {
	return e.db.WithTx(ctx, func(tx gorp.Tx) error { return fn(e.svc.NewWriter(tx)) })
}

func (e *env) retrieve(ctx SpecContext, key secret.Key) secret.Secret {
	GinkgoHelper()
	var s secret.Secret
	Expect(e.svc.NewRetrieve().
		Where(secret.MatchKeys(key)).
		Entry(&s).
		Exec(ctx, e.db)).To(Succeed())
	return s
}

var _ = Describe("Service", func() {
	Describe("OpenService", func() {
		It("Should read values saved before opening again with the same key",
			func(ctx SpecContext) {
				e := newStore(ctx)
				key := secret.NewEncryptionKey()
				e.svc = MustSucceed(openService(ctx, e, key))
				s := e.createStored(ctx, "Password", "hunter2")
				Expect(e.svc.Close()).To(Succeed())
				e.svc = MustOpen(openService(ctx, e, key))
				rk := rack.Key(1).ResourceID()
				g := MustSucceed(e.svc.Grant(ctx, e.db, rk, s.Base().Key))
				Expect(e.svc.Resolve(ctx, rk, g, s.Base().Key)).To(Equal(
					map[secret.Key][]byte{s.Base().Key: []byte("hunter2")},
				))
			},
		)
		It("Should refuse an encryption key that cannot decrypt the stored secrets",
			func(ctx SpecContext) {
				e := newStore(ctx)
				svc := MustSucceed(openService(ctx, e, secret.NewEncryptionKey()))
				Expect(svc.Close()).To(Succeed())
				Expect(openService(ctx, e, secret.NewEncryptionKey())).Error().
					To(MatchError(ContainSubstring(
						"the secrets key does not match the key that encrypted",
					)))
			},
		)
		It("Should require an encryption key", func(ctx SpecContext) {
			Expect(openService(ctx, newStore(ctx), secret.EncryptionKey{})).Error().
				To(MatchError(ContainSubstring("encryption_key: required")))
		})
	})

	Describe("NewRetrieve", func() {
		It(
			"Should retrieve secrets without their plain text values",
			func(ctx SpecContext) {
				e := newEnv(ctx)
				s := e.createStored(ctx, "Password", "hunter2")
				stored := e.retrieve(ctx, s.Base().Key).Variant.(secret.StoredSecret)
				Expect(stored.Name).To(Equal("Password"))
				Expect(stored.Value).ToNot(BeEmpty())
				Expect(string(stored.Value)).ToNot(ContainSubstring("hunter2"))
			},
		)
		It("Should search secrets by name", func(ctx SpecContext) {
			e := newEnv(ctx)
			e.createStored(ctx, "Plant password", "a")
			e.createStored(ctx, "Routing key", "b")
			var found []secret.Secret
			Eventually(func(g Gomega) {
				g.Expect(e.svc.NewRetrieve().
					Search("routing").
					Entries(&found).
					Exec(ctx, e.db)).To(Succeed())
				g.Expect(found).To(HaveLen(1))
			}).Should(Succeed())
			Expect(found[0].Base().Name).To(Equal("Routing key"))
		})
	})

	Describe("Uses", func() {
		It("Should return no uses for a secret that nothing references",
			func(ctx SpecContext) {
				e := newEnv(ctx)
				s := e.createStored(ctx, "Password", "hunter2")
				Expect(e.svc.Uses(ctx, e.db, s.Base().Key)).To(BeEmpty())
			},
		)
		It("Should return each record that references the secret",
			func(ctx SpecContext) {
				e := newEnv(ctx)
				s := e.createStored(ctx, "Password", "hunter2")
				MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, device]{
					DB:   e.db,
					Name: "device",
					References: []gorp.Reference[string, device]{
						gorp.NewKeyReference(
							gorp.KeyReferenceConfig[string, device, secret.Key]{
								Name:     "password",
								Table:    "secret",
								OnDelete: gorp.Restrict,
								Kind:     gorp.Use,
								Targets: func(d *device) []secret.Key {
									return []secret.Key{d.Password}
								},
							},
						),
					},
				}))
				Expect(gorp.NewCreate[string, device]().
					Entry(&device{Key: "d1", Password: s.Base().Key}).
					Exec(ctx, e.db)).To(Succeed())
				Expect(e.svc.Uses(ctx, e.db, s.Base().Key)).
					To(Equal([]resource.ID{{Type: "device", Key: "d1"}}))
			},
		)
	})

	Describe("Changes", func() {
		It(
			"Should return the create, rename, and delete that a commit makes",
			func(ctx SpecContext) {
				e := newEnv(ctx)
				renamed := e.createStored(ctx, "Before", "a")
				deleted := e.createStored(ctx, "Deleted", "b")
				var changes []gorp.EntryChange[secret.Key, secret.Secret]
				DeferCleanup(
					e.db.Observe().OnChange(func(ctx context.Context, c gorp.Commit) {
						for ch := range e.svc.Changes(ctx, c) {
							changes = append(changes, ch)
						}
					}),
				)
				created := secret.Secret{
					Variant: secret.StoredSecret{Base: secret.Base{Name: "Created"}},
				}
				Expect(e.write(ctx, func(w secret.Writer) error {
					if err := w.Set(ctx, &created, []byte("c")); err != nil {
						return err
					}
					if err := w.Rename(ctx, renamed.Base().Key, "After"); err != nil {
						return err
					}
					return w.Delete(ctx, deleted.Base().Key)
				})).To(Succeed())
				kinds := make(map[secret.Key]gorp.ChangeKind, len(changes))
				names := make(map[secret.Key]string, len(changes))
				for _, ch := range changes {
					kinds[ch.Key], names[ch.Key] = ch.Kind, ch.Value.Base().Name
				}
				Expect(kinds).To(Equal(map[secret.Key]gorp.ChangeKind{
					created.Base().Key: gorp.ChangeCreate,
					renamed.Base().Key: gorp.ChangeUpdate,
					deleted.Base().Key: gorp.ChangeDelete,
				}))
				Expect(names[created.Base().Key]).To(Equal("Created"))
				Expect(names[renamed.Base().Key]).To(Equal("After"))
			},
		)
	})

	Describe("Resources", func() {
		It("Should name each secret by its name", func(ctx SpecContext) {
			e := newEnv(ctx)
			s := e.createStored(ctx, "Password", "hunter2")
			Expect(
				e.svc.Resources().Retrieve(ctx, e.db, []resource.ID{s.ResourceID()}),
			).
				To(Equal([]resource.Resource{{ID: s.ResourceID(), Name: "Password"}}))
		})
		It("Should convert between keys and resource IDs", func() {
			s := secret.Secret{Variant: secret.FileSecret{Base: secret.Base{
				Key: uuid.New(),
			}}}
			Expect(secret.ResourceID(s.Base().Key)).To(Equal(s.ResourceID()))
			Expect(secret.ResourceIDs([]secret.Key{s.Base().Key})).
				To(Equal([]resource.ID{s.ResourceID()}))
			Expect(secret.ResourceIDsFromSecrets([]secret.Secret{s})).
				To(Equal([]resource.ID{s.ResourceID()}))
		})
	})
})
