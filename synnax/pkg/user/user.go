// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package user

import (
	"github.com/google/uuid"
)

type User struct {
	Key      uuid.UUID `json:"key"`
	Username string    `json:"username"`
}

func (u User) GorpKey() uuid.UUID { return u.Key }

func (u User) SetOptions() []interface{} { return nil }
