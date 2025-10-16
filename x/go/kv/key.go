// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv

import (
	"bytes"
	"encoding/binary"

	"github.com/synnaxlabs/x/errors"
)

func CompositeKey(elems ...any) ([]byte, error) {
	b := new(bytes.Buffer)
	c := errors.NewCatcher()
	for _, e := range elems {
		switch e := e.(type) {
		case string:
			c.Exec(func() error {
				_, err := b.WriteString(e)
				return err
			})
		default:
			c.Exec(func() error { return binary.Write(b, binary.LittleEndian, e) })
		}
	}
	return b.Bytes(), c.Error()
}
