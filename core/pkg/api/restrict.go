// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api

import (
	"context"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// restrictMiddleware rewrites the error of a delete that a restrict reference refuses,
// so it names the records involved instead of their tables and keys.
func restrictMiddleware(
	res *resource.Service,
	tasks *task.Service,
) freighter.Middleware {
	return freighter.MiddlewareFunc(func(
		ctx freighter.Context,
		next freighter.Next,
	) (freighter.Context, error) {
		oCtx, err := next(ctx)
		if err == nil {
			return oCtx, nil
		}
		if described := describeRestrict(ctx, res, tasks, err); described != nil {
			return oCtx, described
		}
		return oCtx, err
	})
}

// describeRestrict rewrites the gorp.RestrictError in err, alone or under a
// validate.PathError, to name the records involved. It returns nil when err holds no
// gorp.RestrictError.
func describeRestrict(
	ctx context.Context,
	res *resource.Service,
	tasks *task.Service,
	err error,
) error {
	var pathErr validate.PathError
	if errors.As(err, &pathErr) {
		described := describeRestrict(ctx, res, tasks, pathErr.Err)
		if described == nil {
			return nil
		}
		pathErr.Err = described
		return pathErr
	}
	var restrict gorp.RestrictError
	if !errors.As(err, &restrict) {
		return nil
	}
	target := resource.ID{
		Type: resource.Type(restrict.Target.Table),
		Key:  restrict.Target.Key,
	}
	holder, ofTask := tasks.ConfigTask(restrict.Holder)
	if !ofTask {
		holder = resource.ID{
			Type: resource.Type(restrict.Holder.Table),
			Key:  restrict.Holder.Key,
		}
	}
	found, rErr := res.Retrieve(ctx, nil, target, holder)
	if rErr != nil {
		return rErr
	}
	names := make(map[resource.ID]string, len(found))
	for _, r := range found {
		names[r.ID] = r.Name
	}
	// A concurrent delete can remove a record between the refusal and the lookup.
	name := func(id resource.ID) string {
		if n, ok := names[id]; ok {
			return n
		}
		return id.Key
	}
	msg := "cannot delete " + string(target.Type) + " " + name(target) + " because " +
		string(holder.Type) + " " + name(holder) + " uses it"
	if !ofTask {
		msg += " as its " + restrict.Reference
	}
	return errors.Wrap(validate.ErrValidation, msg)
}
