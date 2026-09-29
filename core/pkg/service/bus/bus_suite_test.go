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
	"github.com/synnaxlabs/synnax/pkg/service/library"
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

func binaryField(name string, startBit uint16) library.Field {
	return library.Field{Variant: library.BinaryField{
		BaseField: library.BaseField{Key: uuid.New(), Name: name},
		StartBit:  startBit,
		BitLength: 8,
		ByteOrder: library.ByteOrderLittleEndian,
	}}
}

func fieldKey(f library.Field) library.FieldKey {
	return f.Variant.(library.BinaryField).Key
}

func canMessage(name string, id uint32, fields ...library.Field) library.MessageEntry {
	return library.MessageEntry{
		Key: uuid.New(), Name: name,
		Identifier: &library.Identifier{Variant: library.CanIdentifier{ID: id}},
		Format:     library.FormatBinary,
		Fields:     fields,
	}
}

// createLibrary stores l and defines the ontology resources of l and the task.
func createLibrary(ctx context.Context, l *library.Library, task uuid.UUID) {
	GinkgoHelper()
	l.Key = uuid.New()
	Expect(gorp.NewCreate[library.Key, library.Library]().
		Entry(l).
		Exec(ctx, tx)).To(Succeed())
	Expect(otg.NewWriter(tx).DefineResources(
		ctx,
		library.OntologyID(l.Key),
		ontology.ID{Type: ontology.ResourceTypeTask, Key: task.String()},
	)).To(Succeed())
}
