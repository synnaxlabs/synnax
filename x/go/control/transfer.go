// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package control

import (
	"fmt"

	"go.uber.org/zap"
)

// Subject is digest-style information about a subject that is attempting to control a
// particular resource.
type Subject struct {
	// Key is the key of the subject. This should be unique when compared to all other
	// subjects attempting to control the same resource.
	Key string `json:"key" msgpack:"key"`
	// Name is a pretty name for the subject.
	Name string `json:"name" msgpack:"name"`
}

// String implements fmt.Stringer to nicely print out information about the subject.
func (s Subject) String() string {
	if s.Name != "" {
		return fmt.Sprintf("[%s]<%s>", s.Name, s.Key)
	}
	return fmt.Sprintf("<%s>", s.Key)
}

// State represents the control state of a subject over a resource with a particular
// authority. It is used to indicate the states of several subjects who are contending
// over a particular resource.
type State[R comparable] struct {
	// Resource is the resource under control.
	Resource R `json:"resource" msgpack:"resource"`
	// Subject is the subject controlling (or attempting to control) the resource.
	Subject Subject `json:"subject" msgpack:"subject"`
	// Authority is the authority that the subject has over the resource. A higher
	// authority means a higher precedence over a subject with a lower Authority.
	Authority Authority `json:"authority" msgpack:"authority"`
}

// String implements fmt.Stringer to print out a nice representation of the control
// state.
func (s State[R]) String() string {
	return fmt.Sprintf(
		"%s with authority %v over %v",
		s.Subject,
		s.Authority,
		s.Resource,
	)
}

// Transfer represents a transfer of control over a resource. It is represented as a
// transition from one state to another over the same resource. A transfer between
// resources that are different ill result in a panic when any transfer methods
// are called.
//
// If From is nil, the entity was uncontrolled before the transfer. If To is nil, the
// resource is uncontrolled after the transfer.
//
// If both From and To are nil, no transfer occurred. If both From and To are not nil,
// and From.Subject != To.Subject, a transfer occurred.
type Transfer[R comparable] struct {
	// From is the control state before the transfer. If From is nil, the entity
	// was uncontrolled before the transfer.
	From *State[R]
	// To is the control state after the transfer. If To is nil, the entity is
	// uncontrolled after the transfer.
	To *State[R]
}

func (t Transfer[R]) assertValid() {
	if t.From == nil || t.To == nil {
		return
	}
	if t.From.Resource != t.To.Resource {
		zap.S().DPanicf("transfer must have the same resource in its to and from field. received %s and %s", t.From.Resource, t.To.Resource)
	}
}

// Occurred returns true if a transfer occurred, i.e., one of From or To is not nil and
// From.Subject != To.Subject.
func (t Transfer[R]) Occurred() bool {
	t.assertValid()
	if t.From != nil && t.To != nil {
		return t.From.Subject != t.To.Subject || t.From.Authority != t.To.Authority
	}
	return t.From != nil || t.To != nil
}

// IsTransfer returns true if the control is a transfer between two controlled states.
func (t Transfer[R]) IsTransfer() bool {
	t.assertValid()
	return t.Occurred() && t.To != nil && t.From != nil
}

// IsRelease returns true if the transfer is a release to an uncontrolled state.
func (t Transfer[R]) IsRelease() bool {
	t.assertValid()
	return t.Occurred() && t.To == nil
}

// IsAcquire returns true if the transfer is an acquisition from an uncontrolled state.
func (t Transfer[R]) IsAcquire() bool {
	t.assertValid()
	return t.Occurred() && t.From == nil
}

// String implements fmt.Stringer to return a nicely formatted string representation of
// the control transfer.
func (t Transfer[R]) String() string {
	t.assertValid()
	if !t.Occurred() {
		return "no transfer occurred"
	}
	if t.IsAcquire() {
		return fmt.Sprintf("%s(%v) acquired %v", t.To.Subject, t.To.Authority, t.To.Resource)
	}
	if t.IsRelease() {
		return fmt.Sprintf("%s(%v) released %v", t.From.Subject, t.From.Authority, t.From.Resource)
	}
	return fmt.Sprintf("transfer over %v from %s(%v) to %s(%v)",
		t.From.Resource,
		t.From.Subject,
		t.From.Authority,
		t.To.Subject,
		t.To.Authority,
	)
}
