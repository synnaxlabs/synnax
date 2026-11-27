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
	"sync"

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
// its subject cannot read, and without the time series that only those samples need.
type recordSender struct {
	freighter.StreamSender[StreamerResponse]
	ctx     context.Context
	subject resource.ID
	service *Service
	// held holds each policy that the subject's roles held at a check since the stream
	// opened.
	held set.Set[resource.ID]
	// applied is the Applied of the responses that Send dropped, which the next
	// response it sends carries.
	applied int
	// keys holds the keys of the request that the responses answer.
	keys streamKeys
	mu   struct {
		sync.Mutex
		// staged holds the keys of the later requests that list keys, which the relay
		// applies in order.
		staged []streamKeys
	}
}

// streamKeys holds the channels of a stream request and the untimed ones among them.
type streamKeys struct{ keys, untimed set.Set[channel.Key] }

func newStreamKeys(req StreamerRequest) streamKeys {
	return streamKeys{keys: set.New(req.Keys...), untimed: set.New(req.UntimedKeys...)}
}

func (s *Service) newRecordSender(
	ctx context.Context,
	subject resource.ID,
	sender freighter.StreamSender[StreamerResponse],
	open StreamerRequest,
) (*recordSender, error) {
	r := &recordSender{
		StreamSender: sender,
		ctx:          ctx,
		subject:      subject,
		service:      s,
		held:         make(set.Set[resource.ID]),
		keys:         newStreamKeys(open),
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
// no samples is not sent, and the next response that Send sends carries its Applied.
func (s *recordSender) Send(res StreamerResponse) error {
	s.apply(res.Applied)
	if !s.carriesRecords(res.Frame) {
		return s.send(res)
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
	res.Frame = frame.NewMulti(keys, series).KeepTimed(s.keys.keys, s.keys.untimed)
	if res.Frame.Empty() {
		s.applied += res.Applied
		return nil
	}
	return s.send(res)
}

// stage holds the keys of req, a request that lists keys, until the response whose
// Applied counts it.
func (s *recordSender) stage(req StreamerRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mu.staged = append(s.mu.staged, newStreamKeys(req))
}

// apply makes the keys of the oldest n staged requests the keys of the responses.
func (s *recordSender) apply(n int) {
	if n == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys = s.mu.staged[n-1]
	s.mu.staged = s.mu.staged[n:]
}

// send sends res with the Applied of the responses that Send dropped before it.
func (s *recordSender) send(res StreamerResponse) error {
	res.Applied += s.applied
	s.applied = 0
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
		Clock:             series.Clock,
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
