// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil provides helpers for constructing gorp databases in tests.
package testutil

import (
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
)

// OpenGorpMsgpackDB returns a gorp.DB over a fresh in-memory key-value store that
// encodes entries with MessagePack, so entry types do not need to implement
// orc.SelfCodec. The caller must close the returned DB.
func OpenGorpMsgpackDB() *gorp.DB {
	return gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec))
}
