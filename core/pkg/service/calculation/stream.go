// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation

import (
	"encoding/json/v2"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// StreamCheck returns what a subject receives of a sample of a change channel of
// calculations, given whether it can retrieve each resource: the sample when send is
// true and replacement is nil, another sample when replacement is not nil, and nothing
// when send is false.
type StreamCheck = func(allowed func(resource.ID) bool) (replacement []byte, send bool)

// Readable returns the check of a change to c that only a subject who reads c
// receives. It finds the channels of c once, so a check of many subjects reads them
// once.
func (s *Service) Readable(c Calculation) StreamCheck {
	channels := s.Channels(c)
	return func(allowed func(resource.ID) bool) ([]byte, bool) {
		_, denied := s.denied(c, channels, allowed)
		return nil, !denied
	}
}

// ReadableIn returns Readable of the calculation with key as commit c left it. It
// refuses every subject when c deleted the calculation or did not change it.
func (s *Service) ReadableIn(c gorp.Commit, key Key) StreamCheck {
	for ch := range s.Changes(c) {
		if ch.Key == key && ch.Kind != gorp.ChangeDelete {
			return s.Readable(ch.Value)
		}
	}
	return refuseAll
}

// SetCheck returns the check of the set of c. A subject who reads c receives the set.
// A subject who does not, but can retrieve the output of c, receives the HiddenOutput
// of the output in its place.
func (s *Service) SetCheck(c Calculation) (StreamCheck, error) {
	hidden, err := json.Marshal(HiddenOutput{Output: c.Output, Hidden: true})
	if err != nil {
		return nil, err
	}
	output := channel.ResourceID(c.Output)
	channels := s.Channels(c)
	return func(allowed func(resource.ID) bool) ([]byte, bool) {
		if _, denied := s.denied(c, channels, allowed); !denied {
			return nil, true
		}
		if allowed(output) {
			return hidden, true
		}
		return nil, false
	}, nil
}

// DeleteSet returns the sample of the set channel that the delete in commit c of the
// calculation with key sends, with its check. A subject who read the calculation learns
// of the delete from the delete channel. A subject who held the HiddenOutput of its
// output receives this sample, a HiddenOutput that no calculation writes the output.
// It returns false when c did not delete the calculation.
func (s *Service) DeleteSet(c gorp.Commit, key Key) ([]byte, StreamCheck, bool) {
	formatted := gorp.FormatKey(key)
	for _, ch := range c.Changes {
		if ch.Record.Table != tableName || ch.Record.Key != formatted ||
			ch.Kind != gorp.ChangeDelete {
			continue
		}
		calc, channels, out, ok := removedLinks(key, ch.Removed)
		if !ok {
			return nil, nil, false
		}
		sample, err := json.Marshal(HiddenOutput{Output: out, Hidden: false})
		if err != nil {
			return nil, nil, false
		}
		output := channel.ResourceID(out)
		return sample, func(allowed func(resource.ID) bool) ([]byte, bool) {
			if _, denied := refused(calc, channels, allowed); !denied {
				return nil, false
			}
			return nil, allowed(output)
		}, true
	}
	return nil, nil, false
}

// removedLinks returns, from the links that the delete of the calculation with key
// removed, its resource, the channels that it linked, and its output. It returns false
// when the links hold no output.
func removedLinks(
	key Key,
	links []gorp.Link,
) (resource.ID, []resource.ID, channel.Key, bool) {
	var (
		channels = make([]resource.ID, 0, len(links))
		output   channel.Key
		found    bool
	)
	for _, l := range links {
		channels = append(channels, resource.ID{
			Type: resource.Type(l.Target.Table),
			Key:  l.Target.Key,
		})
		if l.Tag != channel.WriteTag {
			continue
		}
		k, err := channel.ParseKey(l.Target.Key)
		if err != nil {
			return resource.ID{}, nil, 0, false
		}
		output, found = k, true
	}
	return ResourceID(key), channels, output, found
}

func refuseAll(func(resource.ID) bool) ([]byte, bool) { return nil, false }
