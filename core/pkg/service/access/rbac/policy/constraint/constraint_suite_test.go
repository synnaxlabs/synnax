// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package constraint_test

import (
	"context"
	"io"
	"iter"
	"slices"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/constraint"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/observe"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/zyn"
)

func TestConstraint(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Constraint Suite")
}

// testService implements ontology.Service for test resource types.
type testService struct {
	observe.Noop[iter.Seq[ontology.Change]]
	resourceType ontology.Type
}

var _ ontology.Service = (*testService)(nil)

var testSchema = zyn.Object(map[string]zyn.Schema{"key": zyn.String()})

func (s *testService) Type() ontology.Type { return s.resourceType }

func (s *testService) Schema() zyn.Schema { return testSchema }

func (s *testService) RetrieveResource(
	_ context.Context,
	key string,
	_ gorp.Tx,
) (ontology.Resource, error) {
	return ontology.NewResource(
		s.Schema(),
		ontology.ID{Key: key, Type: s.resourceType},
		"empty",
		struct{ Key string }{Key: key},
	), nil
}

func (s *testService) OpenNexter(
	context.Context,
) (iter.Seq[ontology.Resource], io.Closer, error) {
	return slices.Values([]ontology.Resource{}), xio.NopCloser, nil
}

var (
	ctx    context.Context
	db     *gorp.DB
	otg    *ontology.Ontology
	tx     gorp.Tx
	params constraint.EnforceParams
)

var _ = BeforeSuite(func() {
	db = gorp.Wrap(memkv.New())
})

var _ = AfterSuite(func() {
	Expect(db.Close()).To(Succeed())
})

var _ = BeforeEach(func() {
	ctx = context.Background()
	otg = MustSucceed(ontology.Open(ctx, ontology.Config{
		DB:           db,
		EnableSearch: config.False(),
	}))
	// Register test services for relationship constraint tests
	otg.RegisterService(&testService{resourceType: "test"})
	otg.RegisterService(&testService{resourceType: "test2"})
	tx = db.OpenTx()
	params = constraint.EnforceParams{Tx: tx, Ontology: otg}
})

var _ = AfterEach(func() {
	Expect(tx.Close()).To(Succeed())
	Expect(otg.Close()).To(Succeed())
})
