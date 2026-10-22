// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bus_test

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
	tx  gorp.Tx
)

var (
	_ = BeforeSuite(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New()))
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	})
	_ = BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
)

func TestBus(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Bus Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

func binaryField(name string, startBit uint16) icd.BinaryField {
	return icd.BinaryField{
		Key: uuid.New(), Name: name,
		StartBit:  startBit,
		BitLength: 8,
		ByteOrder: icd.ByteOrderLittleEndian,
	}
}

func canMessage(
	name string,
	id uint32,
	fields ...icd.BinaryField,
) icd.MessageEntry {
	return icd.MessageEntry{
		Key: uuid.New(), Name: name,
		Payload: icd.Payload{Variant: icd.BinaryPayload{
			Identifier: &icd.Identifier{Variant: icd.CanIdentifier{ID: id}},
			Fields:     fields,
		}},
	}
}

// createICD stores ic and defines the ontology resources of ic and the task.
func createICD(ctx context.Context, ic *icd.ICD, task uuid.UUID) {
	GinkgoHelper()
	ic.Key = uuid.New()
	Expect(gorp.NewCreate[icd.Key, icd.ICD]().
		Entry(ic).
		Exec(ctx, tx)).To(Succeed())
	Expect(otg.NewWriter(tx).DefineResources(
		ctx,
		icd.OntologyID(ic.Key),
		ontology.ID{Type: ontology.ResourceTypeTask, Key: task.String()},
	)).To(Succeed())
}
