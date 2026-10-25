// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
)

var _ gorp.Entry[Key] = Project{}

// GorpKey implements gorp.Entry.
func (p Project) GorpKey() Key { return p.Key }

// SetOptions implements gorp.Entry.
func (Project) SetOptions() []any { return nil }

// OntologyID returns the ontology.ID of the resource.
func (p Project) OntologyID() ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeProject, Key: p.Key.String()}
}
