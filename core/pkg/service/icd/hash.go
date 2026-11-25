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
	"encoding/json/v2"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/content"
)

// HashMessage writes to h the payload of the message of ic with key, so an edit to its
// format changes the hash. When ic lacks the message, it writes only that the message
// is absent.
func HashMessage(h *content.Hash, ic ICD, key MessageKey) error {
	m, ok := lo.Find(ic.Messages, func(m Message) bool { return m.Key == key })
	h.Bool(ok)
	if !ok {
		return nil
	}
	b, err := json.Marshal(m.Payload, json.Deterministic(true))
	if err != nil {
		return err
	}
	h.Bytes(b)
	return nil
}
