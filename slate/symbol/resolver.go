// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

type Resolver interface {
	Resolve(name string) (*Symbol, error)
}

type MapResolver map[string]Symbol

func (m MapResolver) Resolve(name string) (*Symbol, error) {
	if s, ok := m[name]; ok {
		return &s, nil
	}
	return nil, errors.WithStack(query.NotFound)
}
