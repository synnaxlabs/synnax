// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package example

//go:generate jerky embedded
type Address struct {
	// Street is the street address.
	Street string `json:"street"`
	// City is the city name.
	City string `json:"city"`
	// State is the state or province.
	State string `json:"state"`
	// ZipCode is the postal code.
	ZipCode string `json:"zip_code"`
	// Country is the country name.
	Country string `json:"country"`
	// County is the county name (new in v2).
	County string `json:"county"`
}
