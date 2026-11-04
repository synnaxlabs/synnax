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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// RecordOf returns the record that a sample of a channel carries.
type RecordOf func(sample []byte) (ontology.ID, error)

// recordSender sends a stream's responses without the samples of record channels that
// its subject cannot read.
type recordSender struct {
	freighter.StreamSender[StreamerResponse]
	ctx     context.Context
	subject ontology.ID
	service *Service
}

// Send implements freighter.StreamSender. A frame left with no samples is not sent.
func (s recordSender) Send(res StreamerResponse) error {
	if !s.carriesRecords(res.Frame) {
		return s.StreamSender.Send(res)
	}
	var (
		keys   channel.Keys
		series []telem.Series
	)
	for key, sr := range res.Frame.Entries() {
		if recordOf, ok := s.service.records[key]; ok {
			var err error
			if sr, err = s.keepReadable(sr, recordOf); err != nil {
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

// keepReadable returns a copy of the series with only the samples that the subject can
// read. Streams share the series they receive, so it never changes them.
func (s recordSender) keepReadable(
	series telem.Series,
	recordOf RecordOf,
) (telem.Series, error) {
	var kept []byte
	for sample := range series.Samples() {
		record, err := recordOf(sample)
		if err != nil {
			return telem.Series{}, err
		}
		err = s.service.access.NewEnforcer(nil).Enforce(s.ctx, access.Request{
			Subject: s.subject,
			Action:  access.ActionRetrieve,
			Objects: []ontology.ID{record},
		})
		if errors.Is(err, access.ErrDenied) {
			continue
		}
		if err != nil {
			return telem.Series{}, err
		}
		if series.DataType.IsVariable() {
			sample = telem.MarshalVariableSample(sample)
		}
		kept = append(kept, sample...)
	}
	series.Data = kept
	return series, nil
}
