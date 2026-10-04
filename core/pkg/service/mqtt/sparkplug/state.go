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

// StateTopic returns the topic of the STATE messages of the host application hostID.
func StateTopic(hostID string) string {
	return Topic{Type: State, HostID: hostID}.String()
}

// EncodeState returns the JSON payload of a STATE message. A host application
// publishes it retained at quality of service 1. The offline payload in the last will
// and the online payload that follows the connect carry the same timestamp.
func EncodeState(online bool, timestamp telem.TimeStamp) []byte {
	b := strconv.AppendBool([]byte(`{"online":`), online)
	b = append(b, `,"timestamp":`...)
	b = strconv.AppendUint(b, uint64(timestamp/millisecond), 10)
	return append(b, '}')
}
