// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package certificate

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the certificate API service.
type ServiceConfig struct {
	// Certificate stores certificates.
	//
	// [REQUIRED]
	Certificate *certificate.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the certificate API
	// service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Certificate = override.Nil(c.Certificate, other.Certificate)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.certificate")
	v.NotNil("certificate", c.Certificate)
	return v.Error()
}

// Service is the API for certificates.
type Service struct {
	internal *certificate.Service
}

// NewService opens a Service with the given configuration.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{internal: cfg.Certificate}, nil
}

type (
	// Key is the key of a certificate.
	Key = certificate.Key
	// Certificate is a certificate as the API returns it.
	Certificate = certificate.Certificate
)

// Input is a certificate to create or replace.
type Input struct {
	// PrivateKeyFile is the name of the private key file of a file certificate, in the
	// secrets folder of the rack.
	PrivateKeyFile *string `json:"private_key_file" msgpack:"private_key_file"`
	// Details describes the leaf of a file certificate, as the file browser read it.
	Details *certificate.Details `json:"details" msgpack:"details"`
	// Source is where the contents live: stored or file.
	Source secret.Source `json:"source" msgpack:"source"`
	// Name is the name of the certificate.
	Name string `json:"name" msgpack:"name"`
	// Pem is the leaf-first chain of a stored certificate.
	Pem string `json:"pem" msgpack:"pem"`
	// PrivateKeyPem is the PEM private key of a stored certificate. The Core saves it
	// as a secret, and no endpoint returns it.
	PrivateKeyPem string `json:"private_key_pem" msgpack:"private_key_pem"`
	// CertificatePath is the path of a file certificate on the computer that uses it.
	CertificatePath string `json:"certificate_path" msgpack:"certificate_path"`
	// Key selects the certificate to replace. A zero key creates a new certificate.
	Key Key `json:"key" msgpack:"key"`
}

func (in Input) certificate() (Certificate, error) {
	base := certificate.Base{Key: in.Key, Name: in.Name}
	v := validate.New("certificate")
	var c Certificate
	switch in.Source {
	case secret.SourceStored:
		const msg = "must be empty for a stored certificate"
		v.Ternary("certificate_path", in.CertificatePath != "", msg)
		v.Ternary("private_key_file", in.PrivateKeyFile != nil, msg)
		c.Variant = certificate.StoredCertificate{Base: base, Pem: in.Pem}
	case secret.SourceFile:
		v.Ternary("pem", in.Pem != "", "must be empty for a file certificate")
		c.Variant = certificate.FileCertificate{
			Base:            base,
			CertificatePath: in.CertificatePath,
			PrivateKeyFile:  in.PrivateKeyFile,
			Details:         in.Details,
		}
	default:
		v.Ternaryf("source", true, "must be stored or file, not %q", in.Source)
	}
	if err := v.Error(); err != nil {
		return Certificate{}, err
	}
	return c, nil
}

// SetRequest holds the certificates to create or replace.
type SetRequest struct {
	// Certificates are the certificates to create or replace.
	Certificates []Input `json:"certificates" msgpack:"certificates"`
}

// SetResponse holds the saved certificates.
type SetResponse struct {
	// Certificates are the saved certificates, with their keys.
	Certificates []Certificate `json:"certificates" msgpack:"certificates"`
}

// Set creates the certificates, or replaces those whose key exists. A private key is
// saved as a secret that the certificate uses.
func (s *Service) Set(
	ctx context.Context,
	tx gorp.Tx,
	req SetRequest,
) (SetResponse, error) {
	w := s.internal.NewWriter(tx)
	res := SetResponse{Certificates: make([]Certificate, len(req.Certificates))}
	for i, in := range req.Certificates {
		c, err := in.certificate()
		if err != nil {
			return SetResponse{}, err
		}
		if err = w.Set(ctx, &c, []byte(in.PrivateKeyPem)); err != nil {
			return SetResponse{}, err
		}
		res.Certificates[i] = c
	}
	return res, nil
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the certificates that the top level of Where
// selects by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, certificate.ResourceID)
}

// Retrieve returns the certificates that the request selects and that the subject can
// retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.internal.NewRetrieve().Entries(&res.Certificates))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err = q.Where(bind.Filter(snap, certificate.Match)).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

// DispatchRequest carries an action sequence to apply to a single certificate.
// DispatchKey is a client-generated identifier for the batch, echoed on the broadcast
// so the sender recognizes its own echo.
type DispatchRequest = actions.DispatchRequest[certificate.Key, certificate.Action]

// Dispatch applies the action sequence to the target certificate atomically.
// Subscribers to the certificate action signals receive the sequence after the
// transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		Apply(ctx, req.Key, req.Actions...)
}

// DeleteRequest selects the certificates to delete.
type DeleteRequest struct {
	// Keys selects the certificates to delete.
	Keys []Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the certificates, the secrets that hold their private keys, and their
// expiry statuses.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}
