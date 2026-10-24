// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package serial_test

import (
	"context"
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	db  *gorp.DB
	otg *ontology.Ontology
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
})

func TestSerial(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Serial Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

// createICD stores an ICD with one message and defines the ontology resource of the
// task that will use it. It returns the ICD and its message.
func createICD(
	ctx context.Context,
	task uuid.UUID,
) (icd.ICD, icd.Message) {
	GinkgoHelper()
	m := icd.Message{
		Key: uuid.New(), Name: "Status",
		Payload: icd.Payload{Variant: icd.BinaryPayload{
			Fields: []icd.BinaryField{
				{Key: uuid.New(), Name: "rpm", BitLength: 16},
			},
		}},
	}
	ic := icd.ICD{
		Key:      uuid.New(),
		Name:     "Bus",
		Messages: []icd.Message{m},
	}
	Expect(gorp.NewCreate[icd.Key, icd.ICD]().
		Entry(&ic).
		Exec(ctx, db)).To(Succeed())
	Expect(otg.NewWriter(nil).DefineResources(
		ctx,
		icd.OntologyID(ic.Key),
		ontology.ID{Type: ontology.ResourceTypeTask, Key: task.String()},
	)).To(Succeed())
	return ic, m
}
