// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package common

import "github.com/synnaxlabs/synnax/pkg/service/ontology"

// ConfigResourceTypes is the closed set of task types whose config records live in
// the ontology. Built-in access policies splice this slice in so a grant on tasks
// also covers their config records.
var ConfigResourceTypes = []ontology.ResourceType{
	ontology.ResourceTypeNiAnalogRead,
	ontology.ResourceTypeNiAnalogWrite,
	ontology.ResourceTypeNiDigitalRead,
	ontology.ResourceTypeNiDigitalWrite,
	ontology.ResourceTypeNiCounterRead,
	ontology.ResourceTypeNiScanner,
	ontology.ResourceTypeOpcRead,
	ontology.ResourceTypeOpcWrite,
	ontology.ResourceTypeOpcScan,
	ontology.ResourceTypeLabjackRead,
	ontology.ResourceTypeLabjackWrite,
	ontology.ResourceTypeLabjackScan,
	ontology.ResourceTypeModbusRead,
	ontology.ResourceTypeModbusWrite,
	ontology.ResourceTypeModbusScan,
	ontology.ResourceTypeEthercatRead,
	ontology.ResourceTypeEthercatWrite,
	ontology.ResourceTypeEthercatScan,
	ontology.ResourceTypeHTTPRead,
	ontology.ResourceTypeHTTPWrite,
	ontology.ResourceTypeHTTPScan,
	ontology.ResourceTypeArcTask,
	ontology.ResourceTypePagerdutyAlert,
	ontology.ResourceTypeRackStatus,
}

// ConfigResourceIDs returns one type-scoped ontology ID for each config resource
// type, in the same order as ConfigResourceTypes.
func ConfigResourceIDs() []ontology.ID {
	ids := make([]ontology.ID, len(ConfigResourceTypes))
	for i, t := range ConfigResourceTypes {
		ids[i] = ontology.ID{Type: t}
	}
	return ids
}
