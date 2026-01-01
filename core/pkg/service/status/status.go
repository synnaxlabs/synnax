// Copyright 2026 Synnax Labs, Inc.
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

type Status[D any] status.Status[D]

var _ gorp.Entry[string] = (*Status[any])(nil)

func (s Status[D]) OntologyID() ontology.ID { return OntologyID(s.Key) }

// GorpKey implements gorp.Entry.
func (s Status[D]) GorpKey() string { return s.Key }

// SetOptions implements gorp.Entry.
func (s Status[D]) SetOptions() []any { return nil }

// CustomTypeName implements types.CustomTypeName to ensure that Status struct does
// not conflict with any other types in gorp.
func (s Status[D]) CustomTypeName() string { return "Status" }
