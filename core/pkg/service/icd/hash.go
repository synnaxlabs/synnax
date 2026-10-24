// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd

import (
	"slices"

	"github.com/synnaxlabs/synnax/pkg/service/library"
	"github.com/synnaxlabs/x/set"
)

// Hash returns the hash of the ICD's messages. The Core stores it in the ICD's Hash
// field, and a writer sends the hash it read so the Core can reject a stale save. A
// rename leaves it unchanged.
func Hash(ic ICD) (string, error) { return library.Hash(ic.Messages) }

// TaskHash returns the hash of the messages with the given keys, the ones a task uses.
// It ignores keys of messages the ICD does not hold. Messages hash in key order, so
// reordering an ICD's messages does not change it.
func TaskHash(ic ICD, keys []MessageKey) (string, error) {
	used := set.New(keys...)
	var messages []Message
	for _, m := range ic.Messages {
		if used.Contains(m.Key) {
			messages = append(messages, m)
		}
	}
	slices.SortFunc(messages, func(a, b Message) int { return a.Key.Compare(b.Key) })
	return library.Hash(messages)
}
