// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mqtt_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Factory", func() {
	Describe("OpenFactory", func() {
		It("Should reject a configuration with no device service", func() {
			Expect(mqtt.OpenFactory(mqtt.FactoryConfig{
				Channel: channelSvc,
				Framer:  framerSvc,
				DB:      db, Status: statusSvc,
			})).Error().To(MatchError("device: must be non-nil"))
		})
	})

	Describe("Close", func() {
		It("Should stop following the edits of broker devices", func() {
			f := MustSucceed(mqtt.OpenFactory(mqtt.FactoryConfig{
				Device:  deviceSvc,
				Channel: channelSvc,
				Framer:  framerSvc,
				DB:      db, Status: statusSvc,
			}))
			Expect(f.Close()).To(Succeed())
		})
	})
})
