// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rbac

import (
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
)

const (
	OntologyType ontology.Type = "policy"
)

var AllowAll = ontology.ID{Type: "allow_all", Key: ""}

func OntologyID(k uuid.UUID) ontology.ID {
	return ontology.ID{Type: OntologyType, Key: k.String()}
}

func OntologyIDs(keys []uuid.UUID) (ids []ontology.ID) {
	return lo.Map(keys, func(k uuid.UUID, _ int) ontology.ID {
		return OntologyID(k)
	})
}

func OntologyIDsFromPolicies(policies []Policy) (ids []ontology.ID) {
	return lo.Map(policies, func(p Policy, _ int) ontology.ID {
		return OntologyID(p.Key)
	})
}
