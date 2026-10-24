// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sparkplug

import (
	"strconv"

	"github.com/synnaxlabs/x/telem"
)

// EncodeState returns the STATE message of the host application hostID, retained at
// quality of service 1. The offline message in the last will and the online message
// that follows the connect carry the same timestamp.
func EncodeState(hostID string, online bool, timestamp telem.TimeStamp) Publication {
	b := strconv.AppendBool([]byte(`{"online":`), online)
	b = append(b, `,"timestamp":`...)
	b = strconv.AppendUint(b, uint64(timestamp/telem.MillisecondTS), 10)
	return Publication{
		Topic:    Topic{Type: State, HostID: hostID}.String(),
		Payload:  append(b, '}'),
		QoS:      1,
		Retained: true,
	}
}
