// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"go/types"

	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
)

// CalculationStatus is a calculated channel status entry.
type CalculationStatus = status.Status[types.Nil]

// CalculationStatusKey returns the status key for the given channel key.
func CalculationStatusKey(key Key) string {
	return OntologyID(key).String()
}

// CalculationStatusFromError builds an error status for a calculated channel.
func CalculationStatusFromError(key Key, name string, msg string, err error) *CalculationStatus {
	return &CalculationStatus{
		Key:         CalculationStatusKey(key),
		Name:        name,
		Variant:     status.VariantError,
		Message:     msg,
		Description: err.Error(),
		Time:        telem.Now(),
	}
}
