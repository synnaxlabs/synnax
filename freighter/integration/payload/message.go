// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package payload

import (
	"fmt"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/binary"
)

type Message struct {
	ID      int    `json:"id" msgpack:"id"`
	Message string `json:"message" msgpack:"message"`
}

var _ binary.StringUnmarshaller = (*Message)(nil)

func (m *Message) UnmarshalString(str string) error {
	_, err := fmt.Sscanf(str, "ID: %d Message: %s", &m.ID, &m.Message)
	return err
}

type ServerStream = freighter.ServerStream[Message, Message]
