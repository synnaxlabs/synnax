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
	"slices"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// recordSender sends a stream's responses without the samples of record channels that
// its subject cannot read.
type recordSender struct {
	freighter.StreamSender[StreamerResponse]
	ctx     context.Context
	subject resource.ID
	service *Service
	// held holds each policy that the subject's roles held at a check since the stream
	// opened.
	held set.Set[resource.ID]
}

func (s *Service) newRecordSender(
	ctx context.Context,
	subject resource.ID,
	sender freighter.StreamSender[StreamerResponse],
) (*recordSender, error) {
	r := &recordSender{
		StreamSender: sender,
		ctx:          ctx,
		subject:      subject,
		service:      s,
		held:         make(set.Set[resource.ID]),
	}
	if _, err := r.allowed(); err != nil {
		return nil, err
	}
	return r, nil
}

// Send implements freighter.StreamSender. It reads the policies of the subject again
// for each frame that holds record samples, so a policy change applies from the next
// such frame. The delete of a policy that the subject's roles held at any such check is
// also sent, because the delete removes what let the subject read it. A frame left with
// no samples is not sent.
func (s *recordSender) Send(res StreamerResponse) error {
	if !s.carriesRecords(res.Frame) {
		return s.StreamSender.Send(res)
	}
	allowed, err := s.allowed()
	if err != nil {
		return err
	}
	deleted := func(id resource.ID) bool { return allowed(id) || s.held.Contains(id) }
	var (
		keys   channel.Keys
		series []telem.Series
	)
	for key, sr := range res.Frame.Entries() {
		if record, ok := s.service.records[key]; ok {
			keep := allowed
			if record.Deleted {
				keep = deleted
			}
			if sr, err = keepAllowed(sr, record, keep); err != nil {
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

// allowed returns the check of the subject's current policies and adds them to held.
func (s *recordSender) allowed() (func(resource.ID) bool, error) {
	var policies []policy.Policy
	if err := s.service.db.WithSnapshot(s.ctx, func(snap gorp.Snapshot) (err error) {
		policies, err = s.service.access.NewEnforcer(snap).Policies(s.ctx, s.subject)
		return err
	}); err != nil {
		return nil, err
	}
	for _, p := range policies {
		s.held.Add(p.ResourceID())
	}
	return rbac.Allows(s.subject, policies, access.ActionRetrieve), nil
}

func (s *recordSender) carriesRecords(fr Frame) bool {
	for key := range fr.Keys() {
		if _, ok := s.service.records[key]; ok {
			return true
		}
	}
	return false
}

// keepAllowed returns a copy of the series with only the samples whose record allowed
// accepts, and for a record with Changed, one of the records that changed it. Such a
// sample is sent as Changed returns it. Streams share the series they receive, so it
// never changes them.
func keepAllowed(
	series telem.Series,
	record signals.Record,
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
		of, err := record.Of(sample)
		if err != nil {
			return telem.Series{}, err
		}
		if !allowed(of) {
			continue
		}
		if record.Changed != nil {
			by, sent, err := record.Changed(sample)
			if err != nil {
				return telem.Series{}, err
			}
			if !slices.ContainsFunc(by, allowed) {
				continue
			}
			sample = sent
		}
		if variable {
			sample = telem.MarshalVariableSample(sample)
		}
		kept.Data = append(kept.Data, sample...)
	}
	return kept, nil
}
