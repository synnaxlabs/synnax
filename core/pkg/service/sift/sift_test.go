// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sift_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/sift"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("SecretUses", func() {
	It("Should report each Sift device that uses a secret", func(ctx SpecContext) {
		key := createAPIKey(ctx, apiKey)
		dev := createDevice(ctx, sift.Properties{URI: "localhost:1", APIKey: key})
		createDevice(ctx, sift.Properties{
			URI:    "localhost:1",
			APIKey: createAPIKey(ctx, apiKey),
		})
		uses := MustSucceed(sift.SecretUses(deviceSvc)(ctx, nil, []secret.Key{key}))
		Expect(uses).To(HaveExactElements(secret.Use{
			Resource: device.OntologyID(dev.Key),
			Name:     dev.Name,
			Racks:    []ontology.ID{rackSvc.EmbeddedKey.OntologyID()},
			Secret:   key,
		}))
	})

	It("Should ignore devices of another make", func(ctx SpecContext) {
		key := createAPIKey(ctx, apiKey)
		dev := device.Device{
			Key:        uuid.New().String(),
			Rack:       rackSvc.EmbeddedKey,
			Location:   "kafka",
			Make:       "kafka",
			Model:      "cluster",
			Name:       "cluster",
			Properties: msgpack.EncodedJSON{"api_key": key.String()},
		}
		Expect(deviceSvc.NewWriter(nil).Create(ctx, &dev)).To(Succeed())
		Expect(sift.SecretUses(deviceSvc)(ctx, nil, []secret.Key{key})).To(BeEmpty())
	})

	It("Should let the rack of a device read its API key", func(ctx SpecContext) {
		key := createAPIKey(ctx, apiKey)
		createDevice(ctx, sift.Properties{URI: "localhost:1", APIKey: key})
		values := MustSucceed(secretSvc.Resolve(
			ctx,
			nil,
			rackSvc.EmbeddedKey.OntologyID(),
			uuid.Nil(),
			key,
		))
		Expect(values).To(HaveKeyWithValue(key, []byte(apiKey)))
	})
})
