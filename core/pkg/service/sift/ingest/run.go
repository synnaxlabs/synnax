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

	runsv2 "github.com/sift-stack/sift/go/gen/sift/runs/v2"
	"github.com/synnaxlabs/x/breaker"
	"github.com/synnaxlabs/x/telem"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Run is a span of the data of an asset that Sift groups and names.
type Run struct {
	// Key identifies the run across calls to SetRun.
	Key string
	// Name is the name of the run in Sift.
	Name string
	// TimeRange is the span of the run.
	TimeRange telem.TimeRange
	// Tags are the tags of the run in Sift.
	Tags []string
}

// SetRun makes r an ad hoc run of the asset in Sift and returns its ID. When a run
// with the key of r exists, SetRun updates its name, span, and tags to match r. SetRun
// retries temporary failures as Upload does. It is safe to call while Upload or Stream
// runs.
func (e *Engine) SetRun(ctx context.Context, r Run) (string, error) {
	brk, err := breaker.NewBreaker(ctx, e.cfg.Retry)
	if err != nil {
		return "", err
	}
	for {
		id, err := e.setRun(ctx, r)
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
		e.cfg.OnRetry(ctx, err)
		if !brk.Wait() {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			return "", err
		}
	}
}

func (e *Engine) setRun(ctx context.Context, r Run) (string, error) {
	res, err := e.runs.ListRuns(ctx, &runsv2.ListRunsRequest{
		PageSize: 1,
		Filter:   fmt.Sprintf("client_key == %q", r.Key),
	})
	if err != nil {
		return "", err
	}
	start := timestamppb.New(time.Unix(0, int64(r.TimeRange.Start)))
	stop := timestamppb.New(time.Unix(0, int64(r.TimeRange.End)))
	if len(res.Runs) > 0 {
		existing := res.Runs[0]
		if existing.Name == r.Name &&
			proto.Equal(existing.StartTime, start) &&
			proto.Equal(existing.StopTime, stop) &&
			slices.Equal(
				slices.Sorted(slices.Values(existing.Tags)),
				slices.Sorted(slices.Values(r.Tags)),
			) {
			return existing.RunId, nil
		}
		if _, err = e.runs.UpdateRun(ctx, &runsv2.UpdateRunRequest{
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
	assetID, err := e.asset(ctx)
	if err != nil {
		return "", err
	}
	created, err := e.runs.CreateAdhocRun(ctx, &runsv2.CreateAdhocRunRequest{
		Name:      r.Name,
		StartTime: start,
		StopTime:  stop,
		AssetIds:  []string{assetID},
		Tags:      r.Tags,
		ClientKey: &r.Key,
	})
	if err != nil {
		return "", err
	}
	return created.Run.RunId, nil
}
