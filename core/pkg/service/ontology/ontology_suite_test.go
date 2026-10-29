// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ontology_test

import (
	"context"
	"iter"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/zyn"
)

func TestOntology(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Ontology Suite", SuiteConfig())
}

type sampleService struct {
	observe.Noop[iter.Seq[ontology.Change]]
}

var _ ontology.Service = (*sampleService)(nil)

type Sample struct{ Key string }

func newSampleType(key string) ontology.ID {
	return ontology.ID{Key: key, Type: ontology.ResourceTypeChannel}
}

var schema = zyn.Object(map[string]zyn.Schema{"key": zyn.String()})

func (s *sampleService) Type() ontology.ResourceType { return ontology.ResourceTypeChannel }

func (s *sampleService) RetrieveResource(
	_ context.Context,
	key string,
	_ gorp.Tx,
) (ontology.Resource, error) {
	return ontology.NewResource(
		schema, newSampleType(key), "empty", Sample{Key: key},
	), nil
}

func (*sampleService) Search(
	context.Context,
	gorp.Tx,
	string,
	int,
) (search.Result[ontology.ID], error) {
	return search.Result[ontology.ID]{}, nil
}

var errRejected = errors.New("rejected by policy")

// rejectPolicy refuses the child or name "rejected" and accepts everything else.
type rejectPolicy struct{}

var _ ontology.Policy = rejectPolicy{}

func (rejectPolicy) ValidateParent(
	_ context.Context,
	_ gorp.Tx,
	_ ontology.ID,
	children ...ontology.ID,
) error {
	for _, child := range children {
		if child.Key == "rejected" {
			return errRejected
		}
	}
	return nil
}

func (rejectPolicy) ValidateName(
	_ context.Context,
	_ gorp.Tx,
	_ ontology.ID,
	name string,
) error {
	if name == "rejected" {
		return errRejected
	}
	return nil
}

var (
	db  *gorp.DB
	otg *ontology.Ontology
	tx  gorp.Tx
)

var _ = BeforeSuite(func() {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	otg = MustOpen(ontology.Open(context.Background(), ontology.Config{DB: db}))
	otg.RegisterService(&sampleService{})
	otg.RegisterPolicy(rejectPolicy{})
})

var _ = BeforeEach(func() {
	tx = DeferClose(db.OpenTx())
})

var _ = ShouldNotLeakGoroutinesPerSpec()
