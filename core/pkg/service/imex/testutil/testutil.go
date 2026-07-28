// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil provides helpers for testing imex exporters and importers.
package testutil

import (
	"encoding/json"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/x/testutil"
)

// WireRoundTrip marshals env to JSON and back, binding the codec Decode needs. Exported
// envelopes carry a body but no codec, so a decode must first pass through the wire.
func WireRoundTrip(env imex.Envelope) imex.Envelope {
	b := testutil.MustSucceed(json.Marshal(env))
	var out imex.Envelope
	gomega.Expect(json.Unmarshal(b, &out)).To(gomega.Succeed())
	return out
}
