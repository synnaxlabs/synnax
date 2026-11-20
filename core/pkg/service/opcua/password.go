// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package opcua

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/x/errors"
)

// deviceMake is the make of an OPC UA device.
const deviceMake = "opc"

// properties holds the fields of OPC UA device properties that the Core reads.
type properties struct {
	Connection struct {
		Password string `json:"password"`
	} `json:"connection"`
}

// PasswordReference returns the reference from each OPC UA device to the secret that
// holds its password. Registered on the device table, it lists the device as a use of
// the secret, stops the secret from being deleted while the device exists, and refuses
// a file secret of a rack other than the device's.
func PasswordReference(
	secrets *secret.Service,
) gorp.Reference[device.Key, device.Device] {
	return secret.NewReference(
		secrets,
		secret.ReferenceConfig[device.Key, device.Device]{
			Name: "properties.connection.password",
			Targets: func(dev *device.Device) []secret.Key {
				if dev.Make != deviceMake {
					return nil
				}
				var props properties
				if err := dev.Properties.Unmarshal(&props); err != nil {
					return nil
				}
				key, err := uuid.Parse(props.Connection.Password)
				if err != nil {
					return nil
				}
				return []secret.Key{key}
			},
			Rack: func(dev *device.Device) rack.Key { return dev.Rack },
		},
	)
}

// PasswordMigration returns the migration that moves the plain text password of each
// OPC UA device into a new secret named "<device name> password", and puts the key of
// the secret in its place. An empty password stays empty.
func PasswordMigration(secrets *secret.Service) gorp.Migration {
	return migrate.New(
		"v62_opcua_password_secrets",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			var devs []device.Device
			if err := gorp.NewRetrieve[device.Key, device.Device]().
				Entries(&devs).
				Exec(ctx, tx); err != nil {
				return err
			}
			moved := make([]device.Device, 0, len(devs))
			for _, dev := range devs {
				if dev.Make != deviceMake {
					continue
				}
				raw, ok := dev.Properties["connection"]
				if !ok {
					continue
				}
				conn, ok := raw.(map[string]any)
				if !ok {
					return errors.Newf(
						"connection of device %s is a %T, not an object",
						dev.Key,
						raw,
					)
				}
				password, _ := conn["password"].(string)
				if password == "" {
					continue
				}
				s := secret.Secret{Variant: secret.StoredSecret{
					Base: secret.Base{Name: dev.Name + " password"},
				}}
				if err := secrets.NewWriter(tx).
					Set(ctx, &s, []byte(password)); err != nil {
					return errors.Wrapf(err, "password of device %s", dev.Key)
				}
				conn["password"] = s.Base().Key.String()
				moved = append(moved, dev)
			}
			return gorp.WrapWriter[device.Key, device.Device](tx).Set(ctx, moved...)
		},
	)
}
