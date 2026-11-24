// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package impact

import (
	"slices"
	"strings"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// channelTable is the name of the table of channels, which is its resource type.
const channelTable = string(resource.TypeChannel)

// Policy is what a heal does with a link that holds the dead key of the channel whose
// path another channel takes.
type Policy uint8

const (
	// PolicyKeep leaves the link on the dead key.
	PolicyKeep Policy = iota + 1
	// PolicyRetarget moves the link onto the channel that takes the path.
	PolicyRetarget
)

// Policies say what a heal does with each reference to the channel table.
type Policies struct {
	// Links maps references to their Policy.
	Links map[gorp.ReferenceID]Policy
	// KeptTables reports whether every reference of a table keeps its links, such as
	// the tables of task configs, so that a new integration never moves its ports.
	KeptTables func(table string) bool
}

// Retargets reports whether a heal moves the link of d onto the channel that takes the
// path. It returns an error when p holds no Policy for the reference of d.
func (p Policies) Retargets(d gorp.Dependent) (bool, error) {
	id := d.ReferenceID()
	policy, ok := p.of(id)
	if !ok {
		return false, errors.Newf("[impact] - no policy for %s", referenceText(id))
	}
	return policy == PolicyRetarget, nil
}

// of returns the Policy of the reference with id, and false when p has none.
func (p Policies) of(id gorp.ReferenceID) (Policy, bool) {
	if p.KeptTables != nil && p.KeptTables(id.Table) {
		return PolicyKeep, true
	}
	policy, ok := p.Links[id]
	return policy, ok
}

// check returns an error naming each row of Links that holds no valid Policy, each row
// in a table that KeptTables keeps, each row for a reference that refs lacks, each
// reference of refs that p leaves out, and each that p retargets but that cannot
// retarget.
func (p Policies) check(refs []gorp.ReferenceInfo) error {
	var invalid, kept, stale, unmapped, fixed []string
	for id, policy := range p.Links {
		if policy != PolicyKeep && policy != PolicyRetarget {
			invalid = append(invalid, referenceText(id))
		}
		if p.KeptTables != nil && p.KeptTables(id.Table) {
			kept = append(kept, referenceText(id))
		}
	}
	if len(invalid) > 0 {
		return errors.Newf("[impact] - no valid policy for %s", sorted(invalid))
	}
	if len(kept) > 0 {
		return errors.Newf(
			"[impact] - rows for %s, whose tables keep every link",
			sorted(kept),
		)
	}
	held := make(set.Set[gorp.ReferenceID], len(refs))
	for _, r := range refs {
		held.Add(r.ID)
	}
	for id := range p.Links {
		if !held.Contains(id) {
			stale = append(stale, referenceText(id))
		}
	}
	if len(stale) > 0 {
		return errors.Newf(
			"[impact] - rows for %s, which the database does not hold",
			sorted(stale),
		)
	}
	for _, r := range refs {
		policy, ok := p.of(r.ID)
		if !ok {
			unmapped = append(unmapped, referenceText(r.ID))
		} else if policy == PolicyRetarget && !r.Retargets {
			fixed = append(fixed, referenceText(r.ID))
		}
	}
	if len(unmapped) > 0 {
		return errors.Newf("[impact] - no policy for %s", sorted(unmapped))
	}
	if len(fixed) > 0 {
		return errors.Newf("[impact] - %s cannot retarget", sorted(fixed))
	}
	return nil
}

func referenceText(id gorp.ReferenceID) string { return id.Table + "." + id.Name }

func sorted(texts []string) string {
	slices.Sort(texts)
	return strings.Join(texts, ", ")
}
