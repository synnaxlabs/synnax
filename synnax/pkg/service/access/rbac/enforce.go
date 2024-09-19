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
	"github.com/synnaxlabs/synnax/pkg/service/access"
)

var _ access.Enforcer = (*Service)(nil)

// Enforce implements the access.Enforcer interface.
func (s *Service) Enforce(ctx context.Context, req access.Request) error {
	var policies []Policy
	if err := s.NewRetriever().Entries(&policies).WhereSubject(req.Subject).Exec(ctx, s.DB); err != nil {
		return err
	}
	if len(policies) == 0 {
		return access.Denied
	}
	if AllowRequest(req, policies) {
		return access.Granted
	}
	return access.Denied
}
