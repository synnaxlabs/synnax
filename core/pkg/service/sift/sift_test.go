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
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/sift"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("APIKeyReference", func() {
	It("Should list each Sift device as a use of its API key", func(ctx SpecContext) {
		key := createAPIKey(ctx, fakeKey)
		dev := createDevice(ctx, sift.Properties{URI: "localhost:1", APIKey: key})
		createDevice(ctx, sift.Properties{
			URI:    "localhost:1",
			APIKey: createAPIKey(ctx, fakeKey),
		})
		Expect(secretSvc.Uses(ctx, db, key)).To(HaveExactElements(dev.ResourceID()))
	})

	It("Should ignore devices of another make", func(ctx SpecContext) {
		key := createAPIKey(ctx, fakeKey)
		dev := device.Device{
			Key:        uuid.New().String(),
			Rack:       rackSvc.EmbeddedKey,
			Location:   "kafka",
			Make:       "kafka",
			Model:      "cluster",
			Name:       "cluster",
			Properties: msgpack.EncodedJSON{"api_key": key.String()},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return deviceSvc.NewWriter(tx).Create(ctx, &dev)
		})).To(Succeed())
		Expect(secretSvc.Uses(ctx, db, key)).To(BeEmpty())
	})

	It("Should ignore a Sift device whose properties do not parse",
		func(ctx SpecContext) {
			key := createAPIKey(ctx, fakeKey)
			dev := device.Device{
				Key:        uuid.New().String(),
				Rack:       rackSvc.EmbeddedKey,
				Location:   "sift",
				Make:       sift.Make,
				Model:      "sift",
				Name:       "sift",
				Properties: msgpack.EncodedJSON{"uri": 5, "api_key": key.String()},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return deviceSvc.NewWriter(tx).Create(ctx, &dev)
			})).To(Succeed())
			deleteAfterSpec(dev)
			Expect(secretSvc.Uses(ctx, db, key)).To(BeEmpty())
		})

	It("Should keep a secret from being deleted while a device uses it",
		func(ctx SpecContext) {
			key := createAPIKey(ctx, fakeKey)
			dev := createDevice(ctx, sift.Properties{URI: "localhost:1", APIKey: key})
			deleteSecret := func(tx gorp.Tx) error {
				return secretSvc.NewWriter(tx).Delete(ctx, key)
			}
			Expect(db.WithTx(ctx, deleteSecret)).To(And(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"device "+dev.Key+" references it through properties.api_key",
				)),
			))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return deviceSvc.NewWriter(tx).Delete(ctx, dev.Key)
			})).To(Succeed())
			Expect(db.WithTx(ctx, deleteSecret)).To(Succeed())
		})

	It("Should reject a device whose API key does not exist", func(ctx SpecContext) {
		key := uuid.New()
		dev := device.Device{
			Key:        uuid.New().String(),
			Rack:       rackSvc.EmbeddedKey,
			Location:   "sift",
			Make:       sift.Make,
			Model:      "sift",
			Name:       "sift",
			Properties: toJSON(sift.Properties{URI: "localhost:1", APIKey: key}),
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return deviceSvc.NewWriter(tx).Create(ctx, &dev)
		})).To(MatchError(ContainSubstring(
			"secret " + key.String() + " does not exist",
		)))
	})

	Describe("File API keys", func() {
		// createFileKey creates a secret that names a file on the rack with key r.
		createFileKey := func(ctx SpecContext, r rack.Key) secret.Key {
			GinkgoHelper()
			s := secret.Secret{Variant: secret.FileSecret{
				Base:     secret.Base{Name: "API key file"},
				FileName: "sift.key",
				Rack:     r,
			}}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return secretSvc.NewWriter(tx).Set(ctx, &s, nil)
			})).To(Succeed())
			return s.Base().Key
		}

		It("Should accept a file API key of the device's rack", func(ctx SpecContext) {
			key := createFileKey(ctx, rackSvc.EmbeddedKey)
			dev := createDevice(ctx, sift.Properties{URI: "localhost:1", APIKey: key})
			Expect(secretSvc.Uses(ctx, db, key)).To(HaveExactElements(dev.ResourceID()))
		})

		It("Should reject a file API key of another rack", func(ctx SpecContext) {
			other := rack.Rack{Name: "Other"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return rackSvc.NewWriter(tx).Create(ctx, &other)
			})).To(Succeed())
			key := createFileKey(ctx, other.Key)
			dev := device.Device{
				Key:        uuid.New().String(),
				Rack:       rackSvc.EmbeddedKey,
				Location:   "sift",
				Make:       sift.Make,
				Model:      "sift",
				Name:       "sift",
				Properties: toJSON(sift.Properties{URI: "localhost:1", APIKey: key}),
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return deviceSvc.NewWriter(tx).Create(ctx, &dev)
			})).To(MatchError(fmt.Sprintf(
				`properties.api_key: secret "API key file" is a file of rack %d, `+
					"not of rack %d: validation error",
				other.Key,
				rackSvc.EmbeddedKey,
			)))
		})
	})
})
