// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/kv"
	. "github.com/synnaxlabs/x/testutil"
)

// SetPreV54Row writes a raw row in the key format releases before v0.54 used:
// msgpack(typeName) + msgpack(key), with an msgpack-encoded value. typeName is the type
// name the writing release reported, which a later rename can move away from. Returns
// the full key so callers can assert the row is gone after normalization.
func SetPreV54Row(
	ctx context.Context,
	kvDB kv.DB,
	typeName string,
	key, value any,
) []byte {
	GinkgoHelper()
	prefix := MustSucceed(msgpack.Codec.Encode(ctx, typeName))
	encodedKey := MustSucceed(msgpack.Codec.Encode(ctx, key))
	fullKey := make([]byte, 0, len(prefix)+len(encodedKey))
	fullKey = append(fullKey, prefix...)
	fullKey = append(fullKey, encodedKey...)
	Expect(kvDB.Set(ctx, fullKey, MustSucceed(msgpack.Codec.Encode(ctx, value)))).
		To(Succeed())
	return fullKey
}
