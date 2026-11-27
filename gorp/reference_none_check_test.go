// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// refNote holds only an optional reference, so it can hold no target at all.
type refNote struct {
	Key    string
	Labels []string
}

func (n refNote) GorpKey() string { return n.Key }

var _ = Describe("A create that holds no target", func() {
	It("Should index the new entry as holding none", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refLabel]{
			DB:   db,
			Name: "label",
		}))
		labels := gorp.NewKeyReference(gorp.KeyReferenceConfig[string, refNote, string]{
			Name:     "labels",
			Table:    "label",
			OnDelete: gorp.Restrict,
			Kind:     gorp.Use,
			Targets:  func(n *refNote) []string { return n.Labels },
		})
		notes := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refNote]{
			DB:         db,
			Name:       "note",
			References: []gorp.Reference[string, refNote]{labels},
		}))
		holdingNone := func() []refNote {
			GinkgoHelper()
			var res []refNote
			Expect(notes.NewRetrieve().
				Where(labels.FilterHolding(false)).
				Entries(&res).
				Exec(ctx, db)).To(Succeed())
			return res
		}
		// The query waits for the index to load, so the create below sees a loaded
		// index that has never held the key.
		Expect(holdingNone()).To(BeEmpty())
		Expect(notes.NewCreate().Entry(&refNote{Key: "n"}).Exec(ctx, db)).To(Succeed())
		Expect(holdingNone()).To(ConsistOf(refNote{Key: "n"}))
	})
})
