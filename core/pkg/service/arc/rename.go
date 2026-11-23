// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"github.com/synnaxlabs/arc/text"
	arcv4 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v4"
)

// RenameChannels applies rename to each channel name in src, read as v0.62 text
// (arcv4.RenameChannels), and reports whether it changed src.
func RenameChannels(src *Source, rename func(string) (string, bool)) bool {
	changed := false
	switch v := src.Variant.(type) {
	case TextSource:
		// An imported Arc carries its text as raw, with no document yet.
		raw := v.Text.Raw
		if len(v.Text.Doc.Inserts) > 0 {
			raw = v.Text.Materialize().Raw
		}
		if renamed := arcv4.RenameChannels(raw, rename); renamed != raw {
			v.Text = text.Text{Doc: text.Create(renamed)}
			src.Variant, changed = v, true
		}
	case GraphSource:
		for i, f := range v.Graph.Functions {
			renamed := arcv4.RenameChannels(f.Body.Raw, rename)
			if renamed != f.Body.Raw {
				v.Graph.Functions[i].Body.Raw = renamed
				src.Variant, changed = v, true
			}
		}
	}
	return changed
}
