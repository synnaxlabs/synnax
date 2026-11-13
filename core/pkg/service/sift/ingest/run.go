// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ingest

import (
	"context"
	"fmt"
	"slices"
	"time"
	"uuid"

	runsv2 "github.com/sift-stack/sift/go/gen/sift/runs/v2"
	"github.com/synnaxlabs/x/breaker"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	xuuid "github.com/synnaxlabs/x/uuid"
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Run is a span of the data of an asset that Sift groups and names.
type Run struct {
	// Key is the key of the range of the run, as a UUID. SetRun keeps one run of each
	// asset for each key.
	Key string
	// Name is the name of the run in Sift.
	Name string
	// TimeRange is the span of the run.
	TimeRange telem.TimeRange
	// Tags are the tags of the run in Sift.
	Tags []string
}

// SetRun makes r an ad hoc run of the asset in Sift and returns its ID. When the asset
// has a run with the key of r, SetRun updates its name, span, and tags to match r.
// SetRun retries temporary failures as Upload does, and fails at once when the key of r
// is not a UUID. It is safe to call while Upload or Stream runs.
func (e *Engine) SetRun(ctx context.Context, r Run) (string, error) {
	rng, err := uuid.Parse(r.Key)
	if err != nil {
		return "", validate.Errorf("run key %q is not a UUID", r.Key)
	}
	key := xuuid.NewV5(rng, e.cfg.Asset).String()
	brk, err := breaker.NewBreaker(ctx, e.cfg.Retry)
	if err != nil {
		return "", err
	}
	for {
		id, err := e.setRun(ctx, r, key)
		if err == nil {
			return id, nil
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		permanent, err := classify(err)
		if permanent {
			return "", err
		}
		e.cfg.OnRunRetry(ctx, err)
		if !brk.Wait() {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			return "", err
		}
	}
}

func (e *Engine) setRun(ctx context.Context, r Run, key string) (string, error) {
	existing, err := e.findRun(ctx, key)
	if err != nil {
		return "", err
	}
	if existing != nil {
		return e.updateRun(ctx, existing, r)
	}
	assetID, err := e.asset(ctx)
	if err != nil {
		return "", err
	}
	start, stop := runSpan(r)
	created, err := e.runs.CreateAdhocRun(ctx, &runsv2.CreateAdhocRunRequest{
		Name:      r.Name,
		StartTime: start,
		StopTime:  stop,
		AssetIds:  []string{assetID},
		Tags:      r.Tags,
		ClientKey: &key,
	})
	if status.Code(err) == codes.AlreadyExists {
		if existing, err = e.findRun(ctx, key); err != nil {
			return "", err
		}
		if existing == nil {
			return "", errors.New("Sift reported a run that it does not list")
		}
		return e.updateRun(ctx, existing, r)
	}
	if err != nil {
		return "", err
	}
	return created.Run.RunId, nil
}

func (e *Engine) findRun(ctx context.Context, key string) (*runsv2.Run, error) {
	res, err := e.runs.ListRuns(ctx, &runsv2.ListRunsRequest{
		PageSize: 1,
		Filter:   fmt.Sprintf("client_key == %q", key),
	})
	if err != nil || len(res.Runs) == 0 {
		return nil, err
	}
	return res.Runs[0], nil
}

func (e *Engine) updateRun(
	ctx context.Context,
	existing *runsv2.Run,
	r Run,
) (string, error) {
	start, stop := runSpan(r)
	if existing.Name == r.Name &&
		proto.Equal(existing.StartTime, start) &&
		proto.Equal(existing.StopTime, stop) &&
		slices.Equal(
			slices.Sorted(slices.Values(existing.Tags)),
			slices.Sorted(slices.Values(r.Tags)),
		) {
		return existing.RunId, nil
	}
	if _, err := e.runs.UpdateRun(ctx, &runsv2.UpdateRunRequest{
		Run: &runsv2.Run{
			RunId:     existing.RunId,
			Name:      r.Name,
			StartTime: start,
			StopTime:  stop,
			Tags:      r.Tags,
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"name", "start_time", "stop_time", "tags"},
		},
	}); err != nil {
		return "", err
	}
	return existing.RunId, nil
}

func runSpan(r Run) (start, stop *timestamppb.Timestamp) {
	return timestamppb.New(time.Unix(0, int64(r.TimeRange.Start))),
		timestamppb.New(time.Unix(0, int64(r.TimeRange.End)))
}
