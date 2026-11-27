// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server

import (
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/patch"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// server.
func (CreatePayload) Handle(Server) (Server, error) {
	return Server{}, actions.CoreOnlyError("an OPC UA server create")
}

// Handle applies the set to the server.
func (p SetPayload) Handle(s Server) (Server, error) {
	return apply(s, patch.Set(p))
}

// Handle applies the remove to the server.
func (p RemovePayload) Handle(s Server) (Server, error) {
	return apply(s, patch.Remove(p))
}

// Handle applies the insert to the server.
func (p InsertPayload) Handle(s Server) (Server, error) {
	return apply(s, patch.Insert(p))
}

// Handle applies the move to the server.
func (p MovePayload) Handle(s Server) (Server, error) {
	return apply(s, patch.Move(p))
}

func apply[A patch.Action](s Server, a A) (Server, error) {
	c, err := patch.Begin(a)
	if err == nil {
		err = s.Patch(c)
	}
	if err != nil {
		return Server{}, err
	}
	return s, nil
}
