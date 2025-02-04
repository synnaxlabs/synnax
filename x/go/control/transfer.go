// Copyright 2025 Synnax Labs, Inc.
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
)

type Subject struct {
	Key  string `json:"key" msgpack:"key"`
	Name string `json:"name" msgpack:"name"`
}

func (s Subject) String() string {
	if s.Name != "" {
		return fmt.Sprintf("[%s]<%s>", s.Name, s.Key)
	}
	return fmt.Sprintf("<%s>", s.Key)
}

type State[R comparable] struct {
	Subject   Subject   `json:"subject" msgpack:"subject"`
	Resource  R         `json:"resource" msgpack:"resource"`
	Authority Authority `json:"authority" msgpack:"authority"`
}

// Transfer represents a transfer of control over an entity. It is represented as a
// transition from one state to another. If From is nil, the entity was uncontrolled
// before the transfer. If To is nil, the entity is uncontrolled after the transfer.
// If both From and To are nil, no transfer occurred. If both From and To are not nil,
// and From.Subject != To.Subject, a transfer occurred.
type Transfer[R comparable] struct {
	// From state represents the state of the gate before the transfer. If From is nil,
	// the entity was uncontrolled before the transfer.
	From *State[R]
	// To state represents the state of the gate after the transfer. If To is nil, the
	// entity is uncontrolled after the transfer.
	To *State[R]
}

// Occurred returns true if a transfer occurred i.e. one of From or To is not nil and
// From.Subject != To.Subject.
func (t Transfer[R]) Occurred() bool {
	if t.From != nil && t.To != nil {
		return t.From.Subject != t.To.Subject || t.From.Authority != t.To.Authority
	}
	return t.From != nil || t.To != nil
}

func (t Transfer[R]) IsTransfer() bool { return t.To != nil && t.From != nil }

// IsRelease returns true if the transfer is a release i.e. To is nil and From is not
// nil.
func (t Transfer[R]) IsRelease() bool { return t.Occurred() && t.To == nil }

func (t Transfer[R]) IsAcquire() bool { return t.Occurred() && t.From == nil }
