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

// Pin is the server certificate that the Driver received from a server whose
// connection trusts the first certificate it receives.
type Pin struct {
	// Endpoint is the endpoint that the Driver connected to.
	Endpoint string
	// Policy is the security policy that the Driver connected with.
	Policy SecurityPolicy
	// Certificate is the certificate that the server presented.
	Certificate certificate.Key
}

// Pin trusts only the server certificate of p on the server stored under key, and
// changes no other field. Pinning the certificate that the server already pins
// succeeds. It returns an error wrapping query.ErrNotFound when no server is stored
// under key, and a validation error when the connection is not secured, its endpoint
// or policy differs from p, it pins another certificate or disables validation, or
// the certificate does not exist.
func (w Writer) Pin(ctx context.Context, key Key, p Pin) error {
	return w.update(ctx, key, func(s Server) (Server, error) {
		channel, ok := securedChannel(s.Connection)
		if !ok {
			return Server{}, pinError("connection.security", "is not secured")
		}
		if s.Connection.Endpoint != p.Endpoint {
			return Server{}, pinError(
				"connection.endpoint",
				"differs from the endpoint that the Driver connected to",
			)
		}
		if channel.Policy != p.Policy {
			return Server{}, pinError(
				"connection.security.policy",
				"differs from the policy that the Driver connected with",
			)
		}
		switch v := channel.ServerTrust.Variant.(type) {
		case FirstUseServerTrust:
		case PinnedServerTrust:
			if v.Certificate == p.Certificate {
				return s, nil
			}
			return Server{}, pinError(
				"connection.security.server_trust",
				"pins another certificate",
			)
		default:
			return Server{}, pinError(
				"connection.security.server_trust",
				"does not trust the first certificate",
			)
		}
		channel.ServerTrust = ServerTrust{
			Variant: PinnedServerTrust{Certificate: p.Certificate},
		}
		if _, ok := s.Connection.Security.Variant.(SignSecurity); ok {
			s.Connection.Security.Variant = SignSecurity{SecuredChannel: channel}
		} else {
			s.Connection.Security.Variant = SignAndEncryptSecurity{
				SecuredChannel: channel,
			}
		}
		return s, nil
	})
}

func pinError(path, msg string) error {
	return validate.PathedError(errors.Wrap(validate.ErrValidation, msg), path)
}
