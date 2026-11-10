// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package tcp_test

import (
	"context"
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var db *gorp.DB

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	MustOpen(icd.OpenTable(ctx, icd.TableConfig{DB: db}))
})

func TestTCP(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service TCP Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

// createICD stores an ICD with one message. It returns the ICD and its message.
func createICD(ctx context.Context) (icd.ICD, icd.Message) {
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
	return ic, m
}
