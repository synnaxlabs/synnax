// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package freighter

import (
	"errors"
	"github.com/synnaxlabs/alamos"
)

var (
	// Unreachable is returned when a target cannot be reached.
	Unreachable = errors.New("[freighter] - target unreachable")
	// SecurityError is returned  when a security error occurs.
	SecurityError = errors.New("[freighter] - security error")
)

// Payload represents a piece of data that can be sent over the freighter.
type Payload = any

type Transport interface {
	alamos.ReportProvider
	Use(...Middleware)
}

type Reporter struct {
	Protocol  string
	Encodings []string
}

func (t Reporter) Report() alamos.Report {
	rep := make(alamos.Report)
	rep["protocol"] = t.Protocol
	rep["encodings"] = t.Encodings
	return rep
}
