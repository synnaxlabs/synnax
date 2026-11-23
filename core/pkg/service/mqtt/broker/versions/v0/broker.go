// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

var _ gorp.Entry[Key] = Broker{}

// CustomTypeName gives brokers a table prefix that cannot collide with the legacy
// device table or another integration's resource.
func (Broker) CustomTypeName() string { return "mqtt_broker" }

// GorpKey returns the key of the broker.
func (b Broker) GorpKey() Key { return b.Key }

// ResourceID returns the resource ID of the broker.
func (b Broker) ResourceID() resource.ID {
	return resource.ID{Type: resource.TypeMqttBroker, Key: b.Key.String()}
}
