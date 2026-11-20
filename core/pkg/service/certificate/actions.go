// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package certificate

import (
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/patch"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// certificate.
func (CreatePayload) Handle(Certificate) (Certificate, error) {
	return Certificate{}, actions.CoreOnlyError("a certificate create")
}

// Handle applies the set to the certificate's name.
func (p SetPayload) Handle(c Certificate) (Certificate, error) {
	return edit(c, patch.Set(p).Apply)
}

// Handle applies the remove to the certificate's name.
func (p RemovePayload) Handle(c Certificate) (Certificate, error) {
	return edit(c, patch.Remove(p).Apply)
}

// Handle applies the insert to the certificate's name.
func (p InsertPayload) Handle(c Certificate) (Certificate, error) {
	return edit(c, patch.Insert(p).Apply)
}

// Handle applies the move to the certificate's name.
func (p MovePayload) Handle(c Certificate) (Certificate, error) {
	return edit(c, patch.Move(p).Apply)
}

// document is the part of a certificate that path actions edit. The chain, the private
// key, and the paths change only through a create, which replaces the certificate.
type document struct {
	Name string `json:"name"`
}

// edit applies a path action to a copy of the certificate's document. A document that
// gains a member it does not declare is rejected.
func edit(c Certificate, apply func(map[string]any) error) (Certificate, error) {
	base := c.Base()
	d, err := patch.Edit(document{Name: base.Name}, apply)
	if err != nil {
		return Certificate{}, err
	}
	base.Name = d.Name
	c.SetBase(base)
	return c, nil
}
