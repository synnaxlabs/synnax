// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrate_test

import (
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	. "github.com/synnaxlabs/gorp/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

type label struct{ Key string }

func (l label) GorpKey() string   { return l.Key }
func (l label) SetOptions() []any { return nil }

type labeled struct {
	Key    int32
	Labels []string
}

func (l labeled) GorpKey() int32    { return l.Key }
func (l labeled) SetOptions() []any { return nil }

var labelsReference = gorp.ReferenceConfig[int32, labeled]{
	Name:     "labels",
	Tables:   []string{"label"},
	OnDelete: gorp.Detach,
	Targets: func(l *labeled) []gorp.Target {
		targets := make([]gorp.Target, len(l.Labels))
		for i, k := range l.Labels {
			targets[i] = gorp.To("label", k)
		}
		return targets
	},
	Detach: func(l *labeled, t gorp.Target) {
		l.Labels = slices.DeleteFunc(
			l.Labels,
			func(k string) bool { return k == t.Key },
		)
	},
}

var _ = Describe("DropDangling", func() {
	It("Should drop the targets that do not exist from every entry", func(
		ctx SpecContext,
	) {
		db := OpenGorpMsgpackDB()
		defer func() { Expect(db.Close()).To(Succeed()) }()
		labels := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[string, label]{
			DB:   db,
			Name: "label",
		}))
		defer func() { Expect(labels.Close()).To(Succeed()) }()
		Expect(labels.NewCreate().Entry(&label{Key: "a"}).Exec(ctx, db)).To(Succeed())
		w := gorp.WrapWriter[int32, labeled](db)
		Expect(w.Set(ctx, labeled{Key: 1, Labels: []string{"a", "gone"}})).To(Succeed())
		Expect(w.Set(ctx, labeled{Key: 2, Labels: []string{"a"}})).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: testNamespace,
			Migrations: []gorp.Migration{migrate.DropDangling(
				"drop_dangling",
				db,
				[]gorp.ReferenceConfig[int32, labeled]{labelsReference},
			)},
		})).To(Succeed())
		var entries []labeled
		Expect(gorp.NewRetrieve[int32, labeled]().Entries(&entries).Exec(ctx, db)).
			To(Succeed())
		Expect(entries).To(ConsistOf(
			labeled{Key: 1, Labels: []string{"a"}},
			labeled{Key: 2, Labels: []string{"a"}},
		))
	})
})
