// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5

import (
	"github.com/synnaxlabs/gorp"
	v4 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v4"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

var (
	_ gorp.Entry[Key]   = Arc{}
	_ imex.BodyExporter = Arc{}
)

// GorpKey implements gorp.Entry.
func (a Arc) GorpKey() Key { return a.Base().Key }

// ResourceID returns the unique resource identifier for the Arc.
func (a Arc) ResourceID() resource.ID {
	return resource.ID{Type: resource.TypeArc, Key: a.Base().Key.String()}
}

// ExportBody implements imex.BodyExporter with the body of v4. The channel lists are
// compile records that hold the keys of this Core, so no file carries them.
func (a Arc) ExportBody() any {
	b := a.Base()
	base := v4.Base{Key: b.Key, Name: b.Name, Source: b.Source, Directory: b.Directory}
	old := v4.Arc{Variant: v4.AutomationArc{Base: base}}
	if _, ok := a.Variant.(ModuleArc); ok {
		old.Variant = v4.ModuleArc{Base: base}
	}
	return old.ExportBody()
}
