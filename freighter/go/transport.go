// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package freighter

import "github.com/synnaxlabs/alamos"

// Payload represents a piece of data that can be sent over the freighter.
type Payload = any

// Transport is a type that can be used to send and receive data over a network.
type Transport interface {
	alamos.ReportProvider
	Use(...Middleware)
}

// Reporter is a type that can be used to report the protocol and encodings of a
// transport.
type Reporter struct {
	// Protocol is the protocol that the transport uses (e.g. HTTP, gRPC, etc.).
	Protocol string
	// Encodings are the encodings that the transport uses (e.g. JSON, Protobuf, etc.).
	Encodings []string
}

var _ alamos.ReportProvider = Reporter{}

// Report returns a report containing the protocol and encodings of the transport.
func (r Reporter) Report() alamos.Report {
	rep := make(alamos.Report)
	rep["protocol"] = r.Protocol
	rep["encodings"] = r.Encodings
	return rep
}
