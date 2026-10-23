// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// usingConfig returns the config of a task that uses the given messages of an ICD,
// with all the fields of each.
func usingConfig(ic icd.ICD, messages ...icd.MessageEntry) msgpack.EncodedJSON {
	uses := make([]any, len(messages))
	for i, m := range messages {
		fields := make([]any, 0)
		for _, b := range m.Payload.FieldBases() {
			fields = append(fields, b.Key.String())
		}
		uses[i] = map[string]any{"message": m.Key.String(), "fields": fields}
	}
	return msgpack.EncodedJSON{"icd": ic.Key.String(), "messages": uses}
}

func createUsingTask(
	ctx context.Context,
	name string,
	ic icd.ICD,
	messages ...icd.MessageEntry,
) task.Task {
	GinkgoHelper()
	t := task.Task{
		Rack:   testRack.Key,
		Name:   name,
		Type:   testTaskType,
		Config: usingConfig(ic, messages...),
	}
	Expect(taskSvc.NewWriter(tx).Create(ctx, &t)).To(Succeed())
	return t
}

func retrieveTask(ctx context.Context, key task.Key) task.Task {
	GinkgoHelper()
	var t task.Task
	Expect(taskSvc.NewRetrieve().
		Where(task.MatchKeys(key)).
		Entry(&t).
		Exec(ctx, tx)).To(Succeed())
	return t
}

func retrieveUsers(ctx context.Context, key icd.Key) []ontology.ID {
	GinkgoHelper()
	var users []ontology.Resource
	Expect(otg.NewRetrieve().
		WhereIDs(icd.OntologyID(key)).
		TraverseTo(ontology.UsersTraverser).
		Entries(&users).
		Exec(ctx, tx)).To(Succeed())
	return ontology.ResourceIDs(users)
}

// retrieveUses returns the uses relationships the task is the source of.
func retrieveUses(ctx context.Context, t task.Task) []ontology.Relationship {
	GinkgoHelper()
	var rels []ontology.Relationship
	Expect(gorp.NewRetrieve[string, ontology.Relationship]().
		Where(gorp.Match(func(_ gorp.Context, rel *ontology.Relationship) (bool, error) {
			return rel.From == t.OntologyID() &&
				rel.Type == ontology.RelationshipTypeUses, nil
		})).
		Entries(&rels).
		Exec(ctx, tx)).To(Succeed())
	return rels
}

var _ = Describe("Task references", func() {
	var (
		ic            icd.ICD
		status, brake icd.MessageEntry
	)
	BeforeEach(func(ctx SpecContext) {
		ic = icd.ICD{
			Name: "Bus",
			Entries: []icd.Entry{
				messageEntry(canMessage("Status", 0x100, binaryField("rpm", 0, 16))),
				messageEntry(
					canMessage("Brake", 0x200, binaryField("pressure", 0, 16)),
				),
			},
		}
		Expect(svc.NewWriter(tx).Create(ctx, &ic)).To(Succeed())
		status, brake = messageOf(ic, 0), messageOf(ic, 1)
	})

	It(
		"Should stamp the hash of the entries a task uses into its config",
		func(ctx SpecContext) {
			t := createUsingTask(ctx, "Decoder", ic, status)
			hash := MustSucceed(icd.TaskHash(ic, []icd.EntryKey{status.Key}))
			Expect(t.Config).To(HaveKeyWithValue("icd_hash", hash))
			Expect(
				retrieveTask(ctx, t.Key).Config,
			).To(HaveKeyWithValue("icd_hash", hash))
		},
	)

	It("Should relate the task to the ICD it uses", func(ctx SpecContext) {
		t := createUsingTask(ctx, "Decoder", ic, status)
		Expect(retrieveUsers(ctx, ic.Key)).To(ConsistOf(t.OntologyID()))
	})

	It(
		"Should move the relationship when the task switches ICDs",
		func(ctx SpecContext) {
			t := createUsingTask(ctx, "Decoder", ic, status)
			other := icd.ICD{Name: "Other"}
			Expect(svc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
			t.Config = msgpack.EncodedJSON{"icd": other.Key.String()}
			Expect(taskSvc.NewWriter(tx).Create(ctx, &t)).To(Succeed())
			Expect(retrieveUsers(ctx, ic.Key)).To(BeEmpty())
			Expect(retrieveUsers(ctx, other.Key)).To(ConsistOf(t.OntologyID()))
		},
	)

	It("Should store a task with no ICD and relate it to none", func(ctx SpecContext) {
		t := task.Task{
			Rack:   testRack.Key,
			Name:   "Draft",
			Type:   testTaskType,
			Config: msgpack.EncodedJSON{},
		}
		Expect(taskSvc.NewWriter(tx).Create(ctx, &t)).To(Succeed())
		Expect(retrieveTask(ctx, t.Key).Config).To(SatisfyAll(
			Not(HaveKey("icd")),
			HaveKeyWithValue("icd_hash", ""),
		))
		Expect(retrieveUses(ctx, t)).To(BeEmpty())
	})

	It(
		"Should remove the relationship when the task stops using an ICD",
		func(ctx SpecContext) {
			t := createUsingTask(ctx, "Decoder", ic, status)
			t.Config = msgpack.EncodedJSON{}
			Expect(taskSvc.NewWriter(tx).Create(ctx, &t)).To(Succeed())
			Expect(retrieveUsers(ctx, ic.Key)).To(BeEmpty())
			Expect(retrieveUses(ctx, t)).To(BeEmpty())
			Expect(
				retrieveTask(ctx, t.Key).Config,
			).To(HaveKeyWithValue("icd_hash", ""))
		},
	)

	It("Should reject a task that references a missing ICD", func(ctx SpecContext) {
		key := uuid.New()
		t := task.Task{
			Rack:   testRack.Key,
			Name:   "Orphan",
			Type:   testTaskType,
			Config: msgpack.EncodedJSON{"icd": key.String()},
		}
		Expect(taskSvc.NewWriter(tx).Create(ctx, &t)).To(MatchError(
			"icd: ICD " + key.String() + " does not exist: validation error",
		))
	})

	It(
		"Should rewrite every task that uses a changed message",
		func(ctx SpecContext) {
			first := createUsingTask(ctx, "First", ic, status)
			second := createUsingTask(ctx, "Second", ic, status, brake)
			ic = setScale(ic, 0, 0.5)
			Expect(svc.NewWriter(tx).Create(ctx, &ic)).To(Succeed())
			for _, t := range []task.Task{first, second} {
				res := retrieveTask(ctx, t.Key)
				used := []icd.EntryKey{status.Key}
				if t.Key == second.Key {
					used = append(used, brake.Key)
				}
				Expect(res.Config).To(HaveKeyWithValue(
					"icd_hash",
					MustSucceed(icd.TaskHash(ic, used)),
				))
				Expect(res.ConfigHash).ToNot(Equal(t.ConfigHash))
			}
		},
	)

	It(
		"Should leave a task untouched when a message it does not use changes",
		func(ctx SpecContext) {
			t := createUsingTask(ctx, "Decoder", ic, status)
			before := stamps
			ic = setScale(ic, 1, 0.5)
			Expect(svc.NewWriter(tx).Create(ctx, &ic)).To(Succeed())
			Expect(stamps).To(Equal(before))
			res := retrieveTask(ctx, t.Key)
			Expect(res.ConfigHash).To(Equal(t.ConfigHash))
			Expect(res.Config["icd_hash"]).To(Equal(t.Config["icd_hash"]))
		},
	)

	It(
		"Should rewrite only the task that uses the changed message",
		func(ctx SpecContext) {
			reader := createUsingTask(ctx, "Reader", ic, status)
			braker := createUsingTask(ctx, "Braker", ic, brake)
			before := stamps
			ic = setScale(ic, 1, 0.5)
			Expect(svc.NewWriter(tx).Create(ctx, &ic)).To(Succeed())
			Expect(stamps).To(Equal(before + 1))
			Expect(retrieveTask(ctx, reader.Key).ConfigHash).
				To(Equal(reader.ConfigHash))
			Expect(retrieveTask(ctx, braker.Key).ConfigHash).
				ToNot(Equal(braker.ConfigHash))
		},
	)

	It(
		"Should leave a task untouched when only the ICD's name changes",
		func(ctx SpecContext) {
			t := createUsingTask(ctx, "Decoder", ic, status)
			before := stamps
			ic.Name = "Renamed"
			Expect(svc.NewWriter(tx).Create(ctx, &ic)).To(Succeed())
			Expect(stamps).To(Equal(before))
			Expect(retrieveTask(ctx, t.Key).ConfigHash).To(Equal(t.ConfigHash))
		},
	)

	It(
		"Should refuse a save that drops a message a task uses",
		func(ctx SpecContext) {
			createUsingTask(ctx, "Decoder", ic, status)
			ic.Entries = ic.Entries[1:]
			Expect(svc.NewWriter(tx).Create(ctx, &ic)).To(MatchError(
				`entries: task "Decoder" uses message "Status", which the ICD no ` +
					"longer has: validation error",
			))
		},
	)

	It(
		"Should refuse a save that drops a field a task uses",
		func(ctx SpecContext) {
			createUsingTask(ctx, "Decoder", ic, status)
			ic.Entries[0].Variant = updateBinary(status, func(p *icd.BinaryPayload) {
				p.Fields = []icd.BinaryField{binaryField("speed", 0, 16)}
			})
			Expect(svc.NewWriter(tx).Create(ctx, &ic)).To(MatchError(
				`entries: task "Decoder" uses field "rpm" of message "Status", ` +
					"which the ICD no longer has: validation error",
			))
		},
	)

	It("Should relate a copied task to the ICD", func(ctx SpecContext) {
		t := createUsingTask(ctx, "Decoder", ic, status)
		copied := MustSucceed(taskSvc.NewWriter(tx).Copy(ctx, t.Key, "Copy", false))
		Expect(retrieveUsers(ctx, ic.Key)).
			To(ConsistOf(t.OntologyID(), copied.OntologyID()))
	})

	It("Should reject deleting an ICD a task uses", func(ctx SpecContext) {
		createUsingTask(ctx, "Decoder", ic, status)
		createUsingTask(ctx, "Sender", ic, brake)
		Expect(svc.NewWriter(tx).Delete(ctx, ic.Key)).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("Bus is used by ")),
			MatchError(ContainSubstring("Decoder")),
			MatchError(ContainSubstring("Sender")),
		))
	})

	It(
		"Should allow deleting an ICD once its tasks are deleted",
		func(ctx SpecContext) {
			t := createUsingTask(ctx, "Decoder", ic, status)
			Expect(taskSvc.NewWriter(tx).Delete(ctx, t.Key, false)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, ic.Key)).To(Succeed())
		},
	)
})
