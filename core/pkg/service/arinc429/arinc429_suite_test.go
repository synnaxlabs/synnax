// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arinc429_test

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
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
})

func TestARINC429(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service ARINC 429 Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

// createLibrary stores a library with one message and defines the ontology resource
// of the task that will use it. It returns the library and its message.
func createLibrary(
	ctx context.Context,
	task uuid.UUID,
) (library.Library, library.MessageEntry) {
	GinkgoHelper()
	m := library.MessageEntry{
		BaseEntry: library.BaseEntry{Key: uuid.New(), Name: "Status"},
		Format:    library.FormatBinary,
		Fields: []library.Field{{Variant: library.BinaryField{
			BaseField: library.BaseField{Key: uuid.New(), Name: "rpm"},
			BitLength: 16,
		}}},
	}
	l := library.Library{
		Key:     uuid.New(),
		Name:    "Bus",
		Entries: []library.Entry{{Variant: m}},
	}
	Expect(gorp.NewCreate[library.Key, library.Library]().
		Entry(&l).
		Exec(ctx, db)).To(Succeed())
	Expect(otg.NewWriter(nil).DefineResources(
		ctx,
		library.OntologyID(l.Key),
		ontology.ID{Type: ontology.ResourceTypeTask, Key: task.String()},
	)).To(Succeed())
	return l, m
}
