// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package library_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/library"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func createUsingTask(ctx context.Context, lib library.Key) task.Task {
	t := task.Task{
		Rack:   testRack.Key,
		Name:   "Decoder",
		Type:   testTaskType,
		Config: msgpack.EncodedJSON{"library": lib.String()},
	}
	Expect(taskSvc.NewWriter(tx).Create(ctx, &t)).To(Succeed())
	return t
}

func retrieveTask(ctx context.Context, key task.Key) task.Task {
	var t task.Task
	Expect(taskSvc.NewRetrieve().
		Where(task.MatchKeys(key)).
		Entry(&t).
		Exec(ctx, tx)).To(Succeed())
	return t
}

func retrieveUsers(ctx context.Context, lib library.Key) []ontology.ID {
	var users []ontology.Resource
	Expect(otg.NewRetrieve().
		WhereIDs(library.OntologyID(lib)).
		TraverseTo(ontology.UsersTraverser).
		Entries(&users).
		Exec(ctx, tx)).To(Succeed())
	return ontology.ResourceIDs(users)
}

var _ = Describe("Task references", func() {
	var lib library.Library
	BeforeEach(func(ctx SpecContext) {
		lib = library.Library{
			Name: "Bus",
			Entries: []library.Entry{
				messageEntry(canMessage("Status", 0x100, binaryField("rpm", 0, 16))),
			},
		}
		Expect(svc.NewWriter(tx).Create(ctx, &lib)).To(Succeed())
	})

	It(
		"Should stamp the library hash into a task config on write",
		func(ctx SpecContext) {
			t := createUsingTask(ctx, lib.Key)
			hash := MustSucceed(library.Hash(lib))
			Expect(t.Config).To(HaveKeyWithValue("library_hash", hash))
			Expect(
				retrieveTask(ctx, t.Key).Config,
			).To(HaveKeyWithValue("library_hash", hash))
		},
	)

	It("Should relate the task to the library it uses", func(ctx SpecContext) {
		t := createUsingTask(ctx, lib.Key)
		Expect(retrieveUsers(ctx, lib.Key)).To(ConsistOf(t.OntologyID()))
	})

	It(
		"Should move the relationship when the task switches libraries",
		func(ctx SpecContext) {
			t := createUsingTask(ctx, lib.Key)
			other := library.Library{Name: "Other"}
			Expect(svc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
			t.Config = msgpack.EncodedJSON{"library": other.Key.String()}
			Expect(taskSvc.NewWriter(tx).Create(ctx, &t)).To(Succeed())
			Expect(retrieveUsers(ctx, lib.Key)).To(BeEmpty())
			Expect(retrieveUsers(ctx, other.Key)).To(ConsistOf(t.OntologyID()))
		},
	)

	It("Should reject a task that references a missing library", func(ctx SpecContext) {
		t := task.Task{
			Rack:   testRack.Key,
			Name:   "Orphan",
			Type:   testTaskType,
			Config: msgpack.EncodedJSON{"library": uuid.New().String()},
		}
		Expect(taskSvc.NewWriter(tx).Create(ctx, &t)).To(MatchError(ContainSubstring(
			"library: library " + t.Config["library"].(string) +
				" does not exist: validation error",
		)))
	})

	It(
		"Should re-stamp every task that uses a library when it changes",
		func(ctx SpecContext) {
			first := createUsingTask(ctx, lib.Key)
			second := createUsingTask(ctx, lib.Key)
			m := messageOf(lib, 0)
			f := binaryFieldOf(m, 0)
			f.Scale = 0.5
			m.Fields[0].Variant = f
			lib.Entries[0].Variant = m
			Expect(svc.NewWriter(tx).Create(ctx, &lib)).To(Succeed())
			hash := MustSucceed(library.Hash(lib))
			for _, t := range []task.Task{first, second} {
				res := retrieveTask(ctx, t.Key)
				Expect(res.Config).To(HaveKeyWithValue("library_hash", hash))
				Expect(res.ConfigHash).ToNot(Equal(t.ConfigHash))
			}
		},
	)

	It(
		"Should leave the task config hash alone when only the name changes",
		func(ctx SpecContext) {
			t := createUsingTask(ctx, lib.Key)
			lib.Name = "Renamed"
			Expect(svc.NewWriter(tx).Create(ctx, &lib)).To(Succeed())
			Expect(retrieveTask(ctx, t.Key).ConfigHash).To(Equal(t.ConfigHash))
		},
	)

	It("Should relate a copied task to the library", func(ctx SpecContext) {
		t := createUsingTask(ctx, lib.Key)
		copied := MustSucceed(taskSvc.NewWriter(tx).Copy(ctx, t.Key, "Copy", false))
		Expect(retrieveUsers(ctx, lib.Key)).
			To(ConsistOf(t.OntologyID(), copied.OntologyID()))
	})

	It("Should reject deleting a library a task uses", func(ctx SpecContext) {
		createUsingTask(ctx, lib.Key)
		Expect(svc.NewWriter(tx).Delete(ctx, lib.Key)).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("is used by Decoder")),
		))
	})

	It(
		"Should allow deleting a library once its tasks are deleted",
		func(ctx SpecContext) {
			t := createUsingTask(ctx, lib.Key)
			Expect(taskSvc.NewWriter(tx).Delete(ctx, t.Key, false)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, lib.Key)).To(Succeed())
		},
	)
})
