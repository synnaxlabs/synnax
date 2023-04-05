// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ontology_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"testing"
)

type emptyService struct{}

const emptyType ontology.Type = "empty"

func newEmptyID(key string) ontology.ID {
	return ontology.ID{Key: key, Type: emptyType}
}

func (s *emptyService) Schema() *ontology.Schema {
	return &ontology.Schema{
		Type: emptyType,
		Fields: map[string]schema.Field{
			"key": {Type: schema.String},
		},
	}
}

func (s *emptyService) RetrieveEntity(key string) (ontology.Entity, error) {
	e := schema.NewEntity(s.Schema(), "empty")
	schema.Set(e, "key", key)
	return e, nil
}

var (
	db  *gorp.DB
	otg *ontology.Ontology
	txn gorp.Writer
)

var _ = BeforeSuite(func() {
	var err error
	db = gorp.Wrap(memkv.New())
	otg, err = ontology.Open(gorp.Wrap(db))
	Expect(err).ToNot(HaveOccurred())
	otg.RegisterService(&emptyService{})
})

var _ = AfterSuite(func() {
	Expect(db.Close()).To(Succeed())
})

var _ = BeforeEach(func() {
	txn = db.BeginWrite()
})

var _ = AfterEach(func() {
	Expect(txn.Close()).To(Succeed())
})

func TestOntology(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ontology Suite")
}
