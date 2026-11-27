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
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Apply replaces the field group that update carries on the server stored under key and
// keeps the other group as stored. It returns an error wrapping query.ErrNotFound when
// no server is stored under key and a validation error when update is invalid.
func (w Writer) Apply(ctx context.Context, key Key, update Update) error {
	switch v := update.Variant.(type) {
	case IdentityUpdate:
		return w.update(ctx, key, func(s Server) (Server, error) {
			s.Rack = v.Rack
			s.Name = v.Name
			s.Connection = v.Connection
			return s, nil
		})
	case ConfigurationUpdate:
		return w.update(ctx, key, func(s Server) (Server, error) {
			s.Reads = v.Reads
			s.Writes = v.Writes
			return s, nil
		})
	default:
		return errors.Wrapf(
			validate.ErrValidation,
			"server update has no field group",
		)
	}
}

// Pin trusts only the server certificate cert on the server stored under key, whose
// connection trusts the first certificate it receives. It changes no other field. It
// returns an error wrapping query.ErrNotFound when no server is stored under key, and
// a validation error when the connection is not secured, does not trust the first
// certificate, or cert does not exist.
func (w Writer) Pin(ctx context.Context, key Key, cert certificate.Key) error {
	return w.update(ctx, key, func(s Server) (Server, error) {
		pinned := ServerTrust{Variant: PinnedServerTrust{Certificate: cert}}
		switch v := s.Connection.Security.Variant.(type) {
		case SignSecurity:
			if err := checkFirstUse(v.ServerTrust); err != nil {
				return Server{}, err
			}
			v.ServerTrust = pinned
			s.Connection.Security.Variant = v
		case SignAndEncryptSecurity:
			if err := checkFirstUse(v.ServerTrust); err != nil {
				return Server{}, err
			}
			v.ServerTrust = pinned
			s.Connection.Security.Variant = v
		default:
			return Server{}, validate.PathedError(
				errors.Wrap(validate.ErrValidation, "connection is not secured"),
				"connection.security",
			)
		}
		return s, nil
	})
}

func checkFirstUse(trust ServerTrust) error {
	if _, ok := trust.Variant.(FirstUseServerTrust); ok {
		return nil
	}
	return validate.PathedError(
		errors.Wrap(validate.ErrValidation, "server trust is not first use"),
		"connection.security.server_trust",
	)
}
