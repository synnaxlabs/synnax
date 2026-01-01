// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package label

import "github.com/synnaxlabs/synnax/pkg/distribution/ontology"

const (
	// LabeledBy indicates that a resource is labeled by another resource. When
	// examining a Relationship of type LabeledBy, the Start field will be the
	// resource that is labeled and the To field will be the resource that is
	// doing the labeling (i.e. Start is LabeledBy To).
	LabeledBy ontology.RelationshipType = "labeled_by"
)
