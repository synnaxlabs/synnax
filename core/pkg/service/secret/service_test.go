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
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// env is a secret service on its own in-memory DB, with uses and a clock that each spec
// controls.
type env struct {
	db   *gorp.DB
	otg  *ontology.Ontology
	svc  *secret.Service
	uses []secret.Use
	now  telem.TimeStamp
}

func (e *env) find(
	_ context.Context,
	_ gorp.Tx,
	keys []secret.Key,
) ([]secret.Use, error) {
	wanted := set.New(keys...)
	var found []secret.Use
	for _, u := range e.uses {
		if wanted.Contains(u.Secret) {
			found = append(found, u)
		}
	}
	return found, nil
}

func openService(
	ctx context.Context,
	e *env,
	key secret.EncryptionKey,
) (*secret.Service, error) {
	GinkgoHelper()
	g := MustOpen(
		group.OpenService(ctx, group.ServiceConfig{DB: e.db, Ontology: e.otg}),
	)
	return secret.OpenService(ctx, secret.ServiceConfig{
		DB:            e.db,
		Ontology:      e.otg,
		Group:         g,
		Uses:          e.find,
		Now:           func() telem.TimeStamp { return e.now },
		EncryptionKey: key,
	})
}

// newStore returns an env with a DB and an ontology, and no service.
func newStore(ctx SpecContext) *env {
	GinkgoHelper()
	e := &env{now: telem.SecondTS * 1000}
	e.db = DeferClose(gorp.Wrap(memkv.New()))
	e.otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: e.db}))
	return e
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
	Expect(e.svc.NewWriter(nil).Create(ctx, &s, []byte(value))).To(Succeed())
	return s
}

func (e *env) retrieve(ctx SpecContext, key secret.Key) secret.Secret {
	GinkgoHelper()
	var s secret.Secret
	Expect(e.svc.NewRetrieve().
		Where(secret.MatchKeys(key)).
		Entry(&s).
		Exec(ctx, nil)).To(Succeed())
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
				rk := rack.Key(1).OntologyID()
				e.uses = []secret.Use{{Secret: s.Base().Key, Racks: []ontology.ID{rk}}}
				Expect(e.svc.Resolve(ctx, nil, rk, "", s.Base().Key)).To(Equal(
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
					Exec(ctx, nil)).To(Succeed())
				g.Expect(found).To(HaveLen(1))
			}).Should(Succeed())
			Expect(found[0].Base().Name).To(Equal("Routing key"))
		})
	})

	Describe("Ontology", func() {
		It("Should define a resource for each secret", func(ctx SpecContext) {
			e := newEnv(ctx)
			s := e.createStored(ctx, "Password", "hunter2")
			var res ontology.Resource
			Expect(e.otg.NewRetrieve().
				WhereIDs(s.OntologyID()).
				Entry(&res).
				Exec(ctx, nil)).To(Succeed())
			Expect(res.Name).To(Equal("Password"))
			Expect(e.svc.Type()).To(Equal(ontology.ResourceTypeSecret))
		})
		It("Should convert between keys and ontology IDs", func() {
			s := secret.Secret{Variant: secret.FileSecret{Base: secret.Base{
				Key: uuid.New(),
			}}}
			Expect(secret.OntologyID(s.Base().Key)).To(Equal(s.OntologyID()))
			Expect(secret.OntologyIDs([]secret.Key{s.Base().Key})).
				To(Equal([]ontology.ID{s.OntologyID()}))
			Expect(secret.OntologyIDsFromSecrets([]secret.Secret{s})).
				To(Equal([]ontology.ID{s.OntologyID()}))
		})
	})
})
