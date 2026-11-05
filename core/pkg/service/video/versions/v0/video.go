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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
)

var _ gorp.Entry[Key] = Video{}

// GorpKey implements gorp.Entry.
func (v Video) GorpKey() Key { return v.Key }

// SetOptions implements gorp.Entry.
func (Video) SetOptions() []any { return nil }

// OntologyID returns the unique ontology identifier for the video view.
func (v Video) OntologyID() ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeVideo, Key: v.Key.String()}
}
