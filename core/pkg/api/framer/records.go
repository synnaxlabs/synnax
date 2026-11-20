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
			if sr, err = keepAllowed(sr, record.Of, keep); err != nil {
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

// referenceResponse returns a frame of key with the target of each change that the
// subject can retrieve along with one of the holders that changed it, and false when it
// would hold none. It leaves held alone, so it can run beside Send.
func (s *recordSender) referenceResponse(
	key channel.Key,
	changes []signals.ReferenceChange,
) (StreamerResponse, bool, error) {
	policies, err := s.policies()
	if err != nil {
		return StreamerResponse{}, false, err
	}
	allowed := rbac.Allows(s.subject, policies, access.ActionRetrieve)
	var data []byte
	for _, c := range changes {
		if allowed(c.Target) && slices.ContainsFunc(c.Holders, allowed) {
			sample := telem.MarshalVariableSample([]byte(c.Target.String()))
			data = append(data, sample...)
		}
	}
	if len(data) == 0 {
		return StreamerResponse{}, false, nil
	}
	series := telem.Series{DataType: telem.StringT, Data: data}
	return StreamerResponse{
		Frame: frame.NewMulti(channel.Keys{key}, []telem.Series{series}),
	}, true, nil
}

// allowed returns the check of the subject's current policies and adds them to held.
func (s *recordSender) allowed() (func(resource.ID) bool, error) {
	policies, err := s.policies()
	if err != nil {
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

// policies returns the current policies of the subject.
func (s *recordSender) policies() ([]policy.Policy, error) {
	var policies []policy.Policy
	err := s.service.db.WithSnapshot(s.ctx, func(snap gorp.Snapshot) (err error) {
		policies, err = s.service.access.NewEnforcer(snap).Policies(s.ctx, s.subject)
		return err
	})
	return policies, err
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
