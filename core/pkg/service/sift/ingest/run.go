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
	"time"

	runsv2 "github.com/sift-stack/sift/go/gen/sift/runs/v2"
	"github.com/synnaxlabs/x/breaker"
	"github.com/synnaxlabs/x/telem"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Run is a span of the data of an asset that Sift groups and names.
type Run struct {
	// Key identifies the run across calls to CreateRun.
	Key string
	// Name is the name of the run in Sift.
	Name string
	// TimeRange is the span of the run.
	TimeRange telem.TimeRange
	// Tags are the tags of the run in Sift.
	Tags []string
}

// CreateRun makes r an ad hoc run of the asset in Sift and returns its ID. When a run
// with the key of r exists, CreateRun returns its ID and changes nothing. CreateRun
// retries temporary failures as Upload does.
func (e *Engine) CreateRun(ctx context.Context, r Run) (string, error) {
	brk, err := breaker.NewBreaker(ctx, e.cfg.Retry)
	if err != nil {
		return "", err
	}
	for {
		id, err := e.createRun(ctx, r)
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
			return "", ctx.Err()
		}
	}
}

func (e *Engine) createRun(ctx context.Context, r Run) (string, error) {
	res, err := e.runs.ListRuns(ctx, &runsv2.ListRunsRequest{
		PageSize: 1,
		Filter:   fmt.Sprintf("client_key == %q", r.Key),
	})
	if err != nil {
		return "", err
	}
	if len(res.Runs) > 0 {
		return res.Runs[0].RunId, nil
	}
	if _, err = e.ingestionConfig(ctx); err != nil {
		return "", err
	}
	created, err := e.runs.CreateAdhocRun(ctx, &runsv2.CreateAdhocRunRequest{
		Name:      r.Name,
		StartTime: timestamppb.New(time.Unix(0, int64(r.TimeRange.Start))),
		StopTime:  timestamppb.New(time.Unix(0, int64(r.TimeRange.End))),
		AssetIds:  []string{e.assetID},
		Tags:      r.Tags,
		ClientKey: &r.Key,
	})
	if err != nil {
		return "", err
	}
	return created.Run.RunId, nil
}
