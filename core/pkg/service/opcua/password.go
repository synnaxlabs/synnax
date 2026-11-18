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
		Password          string `json:"password"`
		ClientCertificate string `json:"client_certificate"`
		ServerCertificate string `json:"server_certificate"`
	} `json:"connection"`
}

// connectionOf returns the connection properties of dev, and false when dev is not an
// OPC UA device or has no connection.
func connectionOf(dev *device.Device) (properties, bool) {
	var props properties
	if dev.Make != deviceMake || dev.Properties.Unmarshal(&props) != nil {
		return properties{}, false
	}
	return props, true
}

// keys returns the keys that values parse to, skipping each value that is not a key.
func keys(values ...string) []uuid.UUID {
	var out []uuid.UUID
	for _, v := range values {
		if key, err := uuid.Parse(v); err == nil {
			out = append(out, key)
		}
	}
	return out
}

// migrateConnections returns a migration that calls update with the connection of each
// OPC UA device that has one, and saves each device for which update returns true.
func migrateConnections(
	name string,
	update func(
		ctx context.Context,
		tx gorp.Tx,
		dev device.Device,
		conn map[string]any,
	) (bool, error),
) gorp.Migration {
	return migrate.New(
		name,
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			var devs []device.Device
			if err := gorp.NewRetrieve[device.Key, device.Device]().
				Entries(&devs).
				Exec(ctx, tx); err != nil {
				return err
			}
			changed := make([]device.Device, 0, len(devs))
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
				updated, err := update(ctx, tx, dev, conn)
				if err != nil {
					return err
				}
				if updated {
					changed = append(changed, dev)
				}
			}
			return gorp.WrapWriter[device.Key, device.Device](tx).Set(ctx, changed...)
		},
	)
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
				props, ok := connectionOf(dev)
				if !ok {
					return nil
				}
				return keys(props.Connection.Password)
			},
			Rack: func(dev *device.Device) rack.Key { return dev.Rack },
		},
	)
}

// PasswordMigration returns the migration that moves the plain text password of each
// OPC UA device into a new secret named "<device name> password", and puts the key of
// the secret in its place. An empty password stays empty.
func PasswordMigration(secrets *secret.Service) gorp.Migration {
	return migrateConnections(
		"v62_opcua_password_secrets",
		func(
			ctx context.Context,
			tx gorp.Tx,
			dev device.Device,
			conn map[string]any,
		) (bool, error) {
			password, _ := conn["password"].(string)
			if password == "" {
				return false, nil
			}
			s := secret.Secret{Variant: secret.StoredSecret{
				Base: secret.Base{Name: dev.Name + " password"},
			}}
			if err := secrets.NewWriter(tx).Set(ctx, &s, []byte(password)); err != nil {
				return false, errors.Wrapf(err, "password of device %s", dev.Key)
			}
			conn["password"] = s.Base().Key.String()
			return true, nil
		},
	)
}
