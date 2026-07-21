// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/gorp"
)

var _ gorp.Entry[Key] = View{}

// GorpKey implements gorp.Entry.
func (v View) GorpKey() Key { return v.Key }

// SetOptions implements gorp.Entry.
func (View) SetOptions() []any { return nil }

// OntologyID returns the unique ontology identifier for the view.
func (v View) OntologyID() ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeView, Key: v.Key.String()}
}
