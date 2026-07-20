// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"fmt"
	"strconv"

	rackv2 "github.com/synnaxlabs/synnax/pkg/service/rack/types/v2"
	"github.com/synnaxlabs/x/gorp"
)

func (k Key) Rack() rackv2.Key { return rackv2.Key(k >> 32) }

func (k Key) LocalKey() uint32 { return uint32(uint64(k) & 0xFFFFFFFF) }

func (k Key) String() string { return strconv.Itoa(int(k)) }

func (k Key) IsValid() bool { return !k.Rack().IsZero() && k.LocalKey() != 0 }

var _ gorp.Entry[Key] = Task{}

func (t Task) GorpKey() Key { return t.Key }

func (t Task) SetOptions() []any { return []any{t.Key.Rack().Node()} }

func (t Task) Rack() rackv2.Key { return t.Key.Rack() }

func (t Task) String() string {
	if t.Name != "" {
		return fmt.Sprintf("[%s]<%s>", t.Name, t.Key)
	}
	return t.Key.String()
}

func (c Command) String() string {
	return fmt.Sprintf("%s (key=%s, task=%s)", c.Type, c.Key, c.Task)
}
