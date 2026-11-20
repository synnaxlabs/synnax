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
	"strings"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// CertificateReference returns the reference from each OPC UA device to its client and
// server certificates. Registered on the device table, it lists the device as a use of
// each certificate, stops a certificate from being deleted while the device exists, and
// refuses a file certificate of a rack other than the device's.
func CertificateReference(
	certs *certificate.Service,
) gorp.Reference[device.Key, device.Device] {
	return certificate.NewReference(
		certs,
		certificate.ReferenceConfig[device.Key, device.Device]{
			Name: "properties.connection.certificates",
			Targets: func(dev *device.Device) []certificate.Key {
				props, ok := connectionOf(dev)
				if !ok {
					return nil
				}
				return keys(
					props.Connection.ClientCertificate,
					props.Connection.ServerCertificate,
				)
			},
			Rack: func(dev *device.Device) rack.Key { return dev.Rack },
		},
	)
}

// pemHeader starts each certificate block of a PEM file.
const pemHeader = "-----BEGIN CERTIFICATE-----"

// CertificateMigration returns the migration that moves the certificates of each OPC UA
// device into the certificate service. A pinned server certificate becomes a stored
// certificate named "<device name> server certificate". A client certificate path
// becomes a file certificate on the device's rack named "<device name> client
// certificate". Its private key path becomes the key file "<certificate key>.pem", with
// the warning "Private key file renamed" on the certificate. An empty field stays
// empty. It clears a server certificate that is not PEM, which had no effect, and one
// that does not parse, which kept the device from connecting. It logs a warning for the
// second.
func CertificateMigration(
	certs *certificate.Service,
	statuses *status.Service,
) gorp.Migration {
	return migrateConnections(
		"v62_opcua_certificates",
		func(
			ctx context.Context,
			tx gorp.Tx,
			ins alamos.Instrumentation,
			dev device.Device,
			conn map[string]any,
		) (bool, error) {
			server, _ := conn["server_certificate"].(string)
			client, _ := conn["client_certificate"].(string)
			key, _ := conn["client_private_key"].(string)
			if server == "" && client == "" && key == "" {
				return false, nil
			}
			w := certs.NewWriter(tx)
			serverKey, err := migrateServerCertificate(ctx, w, ins, dev, server)
			if err != nil {
				return false, err
			}
			conn["server_certificate"] = serverKey
			clientKey, err := migrateClientCertificate(
				ctx,
				w,
				statuses.NewWriter(tx),
				dev,
				client,
				key,
			)
			if err != nil {
				return false, err
			}
			conn["client_certificate"] = clientKey
			delete(conn, "client_private_key")
			return true, nil
		},
	)
}

// migrateServerCertificate stores pem, the pinned server certificate of dev, and
// returns its key. It returns an empty key when pem is not PEM or does not parse.
func migrateServerCertificate(
	ctx context.Context,
	w certificate.Writer,
	ins alamos.Instrumentation,
	dev device.Device,
	pem string,
) (string, error) {
	if !strings.Contains(pem, pemHeader) {
		return "", nil
	}
	c := certificate.Certificate{Variant: certificate.StoredCertificate{
		Base: certificate.Base{Name: dev.Name + " server certificate"},
		Pem:  pem,
	}}
	err := w.Set(ctx, &c, nil)
	if errors.Is(err, validate.ErrValidation) {
		ins.L.Warn(
			"cleared the pinned server certificate of an OPC UA device, since it does "+
				"not parse",
			zap.String("device", dev.Key),
			zap.String("name", dev.Name),
			zap.Error(err),
		)
		return "", nil
	}
	if err != nil {
		return "", errors.Wrapf(err, "server certificate of device %s", dev.Key)
	}
	return c.Base().Key.String(), nil
}

// migrateClientCertificate saves path, the client certificate of dev, as a file
// certificate, and returns its key. When keyPath is not empty, the certificate names a
// key file with a new name, and gets a warning that tells an admin to copy keyPath to
// it. It returns an empty key when path is empty.
func migrateClientCertificate(
	ctx context.Context,
	w certificate.Writer,
	statuses status.Writer,
	dev device.Device,
	path, keyPath string,
) (string, error) {
	if path == "" {
		return "", nil
	}
	key := uuid.New()
	file := certificate.FileCertificate{
		Key:             key,
		Name:            dev.Name + " client certificate",
		CertificatePath: path,
		Rack:            dev.Rack,
	}
	if keyPath != "" {
		file.PrivateKeyFile = new(key.String() + ".pem")
	}
	c := certificate.Certificate{Variant: file}
	if err := w.Set(ctx, &c, nil); err != nil {
		return "", errors.Wrapf(err, "client certificate of device %s", dev.Key)
	}
	if keyPath == "" {
		return key.String(), nil
	}
	renamed := status.Status[any]{
		Owner:   new(c.ResourceID()),
		Name:    file.Name,
		Time:    telem.Now(),
		Variant: status.VariantWarning,
		Message: "Private key file renamed",
		Description: "The Driver reads the private key of this certificate from its " +
			"secrets folder now. Copy " + keyPath + " into the secrets folder of the " +
			"Driver as " + *file.PrivateKeyFile + ".",
	}
	if err := statuses.Set(ctx, &renamed); err != nil {
		return "", err
	}
	return key.String(), nil
}
