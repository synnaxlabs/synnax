// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// PasswordSetRequest sets a user's password.
type PasswordSetRequest struct {
	// Subject is the user whose password to set. Empty means the caller.
	Subject ontology.ID `json:"subject" msgpack:"subject"`
	// Password is the new password.
	Password string `json:"password" msgpack:"password"`
	// CurrentPassword is required when the caller changes its own password.
	CurrentPassword string `json:"current_password" msgpack:"current_password"`
}

// SetPassword sets the password of the caller, or of another user when the caller may
// update that user, and ends every other session the old password opened. A caller
// setting its own password must send the current one. It returns
// [auth.ErrInvalidCredentials] for a wrong current password, a validation error when
// the subject is not a user, and an error wrapping query.ErrNotFound when the user has
// no password.
func (s *Service) SetPassword(
	ctx context.Context,
	tx gorp.Tx,
	req PasswordSetRequest,
) (struct{}, error) {
	caller := GetSubject(ctx)
	subject := lo.Ternary(req.Subject.IsZero(), caller, req.Subject)
	if subject.Type != ontology.ResourceTypeUser {
		return struct{}{}, errors.Wrapf(
			validate.ErrValidation,
			"a %s has no password",
			subject.Type,
		)
	}
	if err := s.enforceOthers(
		ctx,
		caller,
		access.ActionUpdate,
		[]ontology.ID{subject},
	); err != nil {
		return struct{}{}, err
	}
	var current password.Password
	if err := s.password.NewRetrieve().
		Where(password.MatchSubjects(subject)).
		Entry(&current).
		Exec(ctx, tx); err != nil {
		return struct{}{}, err
	}
	var keep []auth.SessionKey
	if subject == caller {
		if _, err := s.password.Authenticate(ctx, proofOf(Credentials{
			Username: current.Username,
			Password: req.CurrentPassword,
		})); err != nil {
			return struct{}{}, err
		}
		keep = []auth.SessionKey{getSession(ctx).Key}
	}
	if err := s.password.NewWriter(tx).
		ChangePassword(ctx, subject, req.Password); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.auth.NewWriter(tx).RevokeCredential(ctx, keep, current.Key)
}
