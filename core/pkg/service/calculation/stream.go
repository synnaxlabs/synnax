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

// streamCheck returns what a subject receives of a sample of a change channel of
// calculations, given whether it can retrieve each resource: the sample when send is
// true and replacement is nil, another sample when replacement is not nil, and nothing
// when send is false.
type streamCheck = func(allowed func(resource.ID) bool) (replacement []byte, send bool)

// SetCheck returns the check of the set of c. A subject who reads c receives the set.
// A subject who does not, but can retrieve the output of c, receives the HiddenOutput
// of the output in its place.
func (s *Service) SetCheck(c Calculation) (streamCheck, error) {
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
		return hidden, allowed(output)
	}, nil
}

// DeleteSets returns the lookup of the sample of the set channel that the delete in c
// of each calculation sends, with its check, by key. A subject who read the calculation
// learns of the delete from the delete channel. A subject who held the HiddenOutput of
// its output receives this sample instead: a HiddenOutput that no calculation writes
// the output.
func (s *Service) DeleteSets(c gorp.Commit) func(key Key) ([]byte, streamCheck, bool) {
	type deleteSet struct {
		sample []byte
		check  streamCheck
	}
	byKey := make(map[string]deleteSet)
	for _, ch := range c.Changes {
		if ch.Record.Table != tableName || ch.Kind != gorp.ChangeDelete {
			continue
		}
		calc, channels, out, ok := removedLinks(ch.Record.Key, ch.Removed)
		if !ok {
			continue
		}
		sample, err := json.Marshal(HiddenOutput{Output: out, Hidden: false})
		if err != nil {
			continue
		}
		output := channel.ResourceID(out)
		byKey[ch.Record.Key] = deleteSet{
			sample: sample,
			check: func(allowed func(resource.ID) bool) ([]byte, bool) {
				if _, denied := refused(calc, channels, allowed); !denied {
					return nil, false
				}
				return nil, allowed(output)
			},
		}
	}
	return func(key Key) ([]byte, streamCheck, bool) {
		d, ok := byKey[gorp.FormatKey(key)]
		return d.sample, d.check, ok
	}
}

// removedLinks returns, from the links that the delete of the calculation with key
// removed, its resource, the channels that it linked, and its output. It returns false
// when the links hold no output.
func removedLinks(
	key string,
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
	return resource.ID{
		Type: resource.TypeCalculation,
		Key:  key,
	}, channels, output, found
}
