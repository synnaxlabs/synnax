// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock

import "github.com/synnaxlabs/alamos"

const (
	protocol = "golang-mock"
	encoding = "in-memory"
)

// reporter describes the mock protocol. Every transport in this package embeds it, so
// a constructor cannot leave a transport without a report.
type reporter struct{}

// Report implements alamos.ReportProvider.
func (reporter) Report() alamos.Report {
	return alamos.Report{"protocol": protocol, "encodings": []string{encoding}}
}
