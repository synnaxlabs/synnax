// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	v1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/resource/rename"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("NameRuleMigration breakers", func() {
	var db *gorp.DB
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })
	channel := func(local v1.LocalKey, name string, group *uuid.UUID) v1.Channel {
		return v1.Channel{
			Name:        name,
			Leaseholder: node.KeyFree,
			LocalKey:    local,
			Virtual:     true,
			Group:       group,
		}
	}
	create := func(ctx SpecContext, channels ...v1.Channel) {
		GinkgoHelper()
		Expect(gorp.NewCreate[v1.Key, v1.Channel]().
			Entries(&channels).
			Exec(ctx, db)).To(Succeed())
	}
	run := func(ctx SpecContext) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return v1.NameRuleMigration.Run(ctx, tx, alamos.Instrumentation{})
		})).To(Succeed())
	}
	stored := func(ctx SpecContext) map[v1.Key]v1.Channel {
		GinkgoHelper()
		var all []v1.Channel
		Expect(gorp.NewRetrieve[v1.Key, v1.Channel]().
			Entries(&all).
			Exec(ctx, db)).To(Succeed())
		out := make(map[v1.Key]v1.Channel, len(all))
		for _, c := range all {
			out[c.Key()] = c
		}
		return out
	}
	legal := func(name string) bool {
		return validate.IsPathSegment(name) && validate.IsNormalName(name)
	}

	It("Should rename a long run of combining marks to a legal name", func(
		ctx SpecContext,
	) {
		zalgo := channel(1, "a"+strings.Repeat("\u0301", 31), nil)
		create(ctx, zalgo)
		run(ctx)
		name := stored(ctx)[zalgo.Key()].Name
		Expect(legal(name)).To(BeTrue(), "%+q", name)
	})

	It("Should keep the first From when a second run finds nothing to fix", func(
		ctx SpecContext,
	) {
		original := "a" + strings.Repeat("\u0301", 31)
		zalgo := channel(1, original, nil)
		create(ctx, zalgo)
		run(ctx)
		run(ctx)
		staged := MustSucceed(rename.Staged(db))
		Expect(staged).To(HaveLen(1))
		Expect(staged[0].From).To(Equal(original))
	})

	It("Should leave no two channels of a group with one name after NFC", func(
		ctx SpecContext,
	) {
		builtIn := channel(1, "e\u0301", nil)
		builtIn.Internal = true
		user := channel(2, "e\u0301", nil)
		create(ctx, builtIn, user)
		run(ctx)
		after := stored(ctx)
		Expect(validate.NormalizeName(after[user.Key()].Name)).
			ToNot(Equal(validate.NormalizeName(after[builtIn.Key()].Name)))
	})

	It("Should not give a rewritten channel the name of a sibling folder", func(
		ctx SpecContext,
	) {
		folder := groupv1.Group{
			Key:   uuid.New(),
			Name:  "a_b",
			Scope: resource.TypeChannel,
		}
		Expect(gorp.NewCreate[groupv1.Key, groupv1.Group]().
			Entry(&folder).
			Exec(ctx, db)).To(Succeed())
		ch := channel(1, "a.b", nil)
		create(ctx, ch)
		run(ctx)
		Expect(stored(ctx)[ch.Key()].Name).ToNot(Equal(folder.Name))
	})
})
