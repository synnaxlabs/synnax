// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package uuid

import (
	"github.com/google/uuid"
	"github.com/samber/lo"
)

type UUID = uuid.UUID

var Nil = uuid.Nil

func Parse(s string) (UUID, error) {
	if s == "" {
		return uuid.Nil, nil
	}
	return uuid.Parse(s)
}

func MustParse(s string) UUID {
	return lo.Must(Parse(s))
}

func New() UUID { return uuid.New() }

func NewString() string { return uuid.NewString() }
