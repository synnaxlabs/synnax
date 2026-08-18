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
	"strconv"

	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
)

// Node returns the node that the rack is leased to.
func (k Key) Node() node.Key { return node.Key(k >> 16) }

// OntologyID returns the unique ontology identifier for the rack.
func (k Key) OntologyID() ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeRack, Key: k.String()}
}

// IsZero returns true if the key is unset.
func (k Key) IsZero() bool { return k == 0 }

// String returns the key formatted as its decimal integer value.
func (k Key) String() string { return strconv.Itoa(int(k)) }
