// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rbac

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/access"
)

var _ access.Enforcer = (*Service)(nil)

// Enforce implements the access.Enforcer interface.
func (s *Service) Enforce(ctx context.Context, req access.Request) error {
	var policies []Policy
	if err := s.NewRetriever().Entries(&policies).WhereSubjects(req.Subject).Exec(ctx, s.DB); err != nil {
		return err
	}
	if allowRequest(req, policies) {
		return access.Granted
	}
	return access.Denied
}
