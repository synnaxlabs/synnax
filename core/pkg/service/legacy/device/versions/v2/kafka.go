// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"context"

	"github.com/synnaxlabs/gorp/migrate"
)

// KafkaSecureMigration renames the tls property of each stored Kafka cluster to secure.
var KafkaSecureMigration = migrate.NewEntry(
	"v62_kafka_tls_secure",
	func(_ context.Context, d Device) (Device, error) {
		if d.Make != "kafka" {
			return d, nil
		}
		tls, ok := d.Properties["tls"]
		if !ok {
			return d, nil
		}
		delete(d.Properties, "tls")
		d.Properties["secure"] = tls
		return d, nil
	},
)
