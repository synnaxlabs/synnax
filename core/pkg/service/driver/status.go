// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package driver

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

// AckCurrentStatus answers the command with the given key by re-sending t's current
// status, for a command that needs no work. It keeps the stored variant, message,
// description, and data, and re-asserts the facts t is authoritative for: running,
// config hash, and rack.
func AckCurrentStatus(
	ctx context.Context,
	svc *status.Service,
	t task.Task,
	cmdKey string,
	running bool,
) error {
	var stat task.Status
	err := status.NewRetrieve[task.StatusDetails](svc).
		Where(status.MatchKeys[task.StatusDetails](t.OntologyID().String())).
		Entry(&stat).
		Exec(ctx, nil)
	if errors.Is(err, query.ErrNotFound) {
		stat.Variant = status.VariantSuccess
	} else if err != nil {
		return err
	}
	details := task.NewStatusDetails(t, running)
	details.Cmd = cmdKey
	details.Data = stat.Details.Data
	stat.Key = t.OntologyID().String()
	stat.Name = t.Name
	stat.Time = telem.Now()
	stat.Details = details
	return status.NewWriter[task.StatusDetails](svc, nil).Set(ctx, &stat)
}
