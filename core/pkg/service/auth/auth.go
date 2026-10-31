// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package auth logs subjects in and keeps their sessions. A subject presents a [Proof];
// the [Authenticator] for its method checks it; the [Service] then issues a session
// token that every later request carries. Each method lives in its own package, such as
// password and apikey.
package auth

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

var _ override.Zeroable = Credentials{}

// IsZero implements override.Zeroable.
func (c Credentials) IsZero() bool { return c == Credentials{} }

func (p Proof) method() (Method, error) {
	switch p.Variant.(type) {
	case PasswordProof:
		return MethodPassword, nil
	case APIKeyProof:
		return MethodAPIKey, nil
	}
	return "", errors.Wrapf(
		validate.ErrValidation,
		"proof has no method: %T",
		p.Variant,
	)
}

// Identity is the subject a proof resolved to and the credential that proved it.
type Identity struct {
	// Subject is the user or rack that logged in.
	Subject ontology.ID
	// Credential is the key of the credential record the proof matched.
	Credential uuid.UUID
	// Version is when the credential's secret was set.
	Version telem.TimeStamp
}

// Authenticator checks proofs for one [Method].
type Authenticator interface {
	// Authenticate checks the proof and returns the identity it proves. It returns
	// [ErrInvalidCredentials] when the proof matches no stored credential.
	Authenticate(ctx context.Context, proof Proof) (Identity, error)
	// DeleteFor removes every credential of the given subjects within tx.
	DeleteFor(ctx context.Context, tx gorp.Tx, subjects ...ontology.ID) error
	// Version returns the [Identity.Version] of the credential within tx. It returns
	// an error wrapping query.ErrNotFound when the authenticator holds no such
	// credential.
	Version(
		ctx context.Context,
		tx gorp.Tx,
		credential uuid.UUID,
	) (telem.TimeStamp, error)
}
