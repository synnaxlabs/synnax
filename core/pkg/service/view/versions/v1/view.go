// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

var _ gorp.Entry[Key] = View{}

// GorpKey returns the view's storage key.
func (v View) GorpKey() Key { return v.Base().Key }

// SetOptions returns no additional storage options.
func (View) SetOptions() []any { return nil }

// ResourceID returns the view's resource identifier.
func (v View) ResourceID() resource.ID {
	return resource.ID{Type: resource.TypeView, Key: v.Base().Key.String()}
}
