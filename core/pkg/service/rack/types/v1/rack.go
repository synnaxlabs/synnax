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
	"strconv"

	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
)

// Node returns the node that the rack is leased to.
func (k Key) Node() node.Key { return node.Key(k >> 16) }

// LocalKey returns unique key for the rack on its leaseholder node.
func (k Key) LocalKey() uint16 { return uint16(uint32(k) & 0xFFFF) }

// OntologyID returns the unique ontology identifier for the rack.
func (k Key) OntologyID() ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeRack, Key: k.String()}
}

// IsZero returns true if the key is invalid i.e. it's Node or LocalKey is zero.
func (k Key) IsZero() bool { return k == 0 }

// String implements fmt.Stringer.
func (k Key) String() string { return strconv.Itoa(int(k)) }
