// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"context"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/telem"
)

// recordSender sends a stream's responses without the samples of record channels that
// its subject cannot read.
type recordSender struct {
	freighter.StreamSender[StreamerResponse]
	ctx     context.Context
	subject resource.ID
	service *Service
}

// Send implements freighter.StreamSender. It reads the policies of the subject again
// for each frame that holds record samples, so a policy change applies from the next
// such frame. A frame left with no samples is not sent.
func (s recordSender) Send(res StreamerResponse) error {
	if !s.carriesRecords(res.Frame) {
		return s.StreamSender.Send(res)
	}
	var allowed func(resource.ID) bool
	err := s.service.db.WithSnapshot(s.ctx, func(snap gorp.Snapshot) (err error) {
		allowed, err = s.service.access.NewEnforcer(snap).Allowed(
			s.ctx,
			s.subject,
			access.ActionRetrieve,
		)
		return err
	})
	if err != nil {
		return err
	}
	var (
		keys   channel.Keys
		series []telem.Series
	)
	for key, sr := range res.Frame.Entries() {
		if recordOf, ok := s.service.records[key]; ok {
			if sr, err = keepAllowed(sr, recordOf, allowed); err != nil {
				return err
			}
			if sr.Size() == 0 {
				continue
			}
		}
		keys = append(keys, key)
		series = append(series, sr)
	}
	if len(keys) == 0 {
		return nil
	}
	res.Frame = frame.NewMulti(keys, series)
	return s.StreamSender.Send(res)
}

func (s recordSender) carriesRecords(fr Frame) bool {
	for key := range fr.Keys() {
		if _, ok := s.service.records[key]; ok {
			return true
		}
	}
	return false
}

// keepAllowed returns a copy of the series with only the samples whose record allowed
// accepts. Streams share the series they receive, so it never changes them.
func keepAllowed(
	series telem.Series,
	recordOf signals.RecordOf,
	allowed func(resource.ID) bool,
) (telem.Series, error) {
	kept := telem.Series{
		TimeRange:         series.TimeRange,
		DataType:          series.DataType,
		Alignment:         series.Alignment,
		AlignmentMultiple: series.AlignmentMultiple,
	}
	variable := series.DataType.IsVariable()
	for sample := range series.Samples() {
		record, err := recordOf(sample)
		if err != nil {
			return telem.Series{}, err
		}
		if !allowed(record) {
			continue
		}
		if variable {
			sample = telem.MarshalVariableSample(sample)
		}
		kept.Data = append(kept.Data, sample...)
	}
	return kept, nil
}
