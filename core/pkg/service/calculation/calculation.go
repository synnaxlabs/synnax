// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation

import "github.com/synnaxlabs/synnax/pkg/service/channel"

// Resets returns the reset channel of each windowed operation of c that has one, in
// the order of the operations.
func Resets(c Calculation) []channel.Key {
	var resets []channel.Key
	for _, op := range c.Operations {
		var w Window
		switch v := op.Variant.(type) {
		case MinOperation:
			w = v.Window
		case MaxOperation:
			w = v.Window
		case AvgOperation:
			w = v.Window
		case DerivativeOperation:
			continue
		}
		if w.Reset != nil {
			resets = append(resets, *w.Reset)
		}
	}
	return resets
}
