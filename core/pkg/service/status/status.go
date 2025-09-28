// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/status"
)

type Status status.Status[any]

var _ gorp.Entry[string] = (*Status)(nil)

func (s Status) OntologyID() ontology.ID { return OntologyID(s.Key) }

// GorpKey implements gorp.Entry.
func (s Status) GorpKey() string { return s.Key }

// SetOptions implements gorp.Entry.
func (s Status) SetOptions() []any { return nil }
