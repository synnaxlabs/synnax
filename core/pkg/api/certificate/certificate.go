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

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
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
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
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
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.certificate")
	v.NotNil("certificate", c.Certificate)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

// Service is the API for certificates.
type Service struct {
	access   *rbac.Service
	internal *certificate.Service
}

// NewService opens a Service with the given configuration.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{access: cfg.RBAC, internal: cfg.Certificate}, nil
}

type (
	// Key is the key of a certificate.
	Key = certificate.Key
	// Certificate is a certificate as the API returns it.
	Certificate = certificate.Certificate
)

// Input is a certificate to create or replace.
type Input struct {
	// PrivateKeyPath is the path of the private key of a file certificate.
	PrivateKeyPath *string `json:"private_key_path,omitzero" msgpack:"private_key_path,omitempty"`
	// Details describes the leaf of a file certificate, as the file browser read it.
	Details *certificate.Details `json:"details,omitzero" msgpack:"details,omitempty"`
	// Source is where the contents live: stored or file.
	Source secret.Source `json:"source" msgpack:"source"`
	// Name is the name of the certificate.
	Name string `json:"name" msgpack:"name"`
	// Pem is the leaf-first chain of a stored certificate.
	Pem string `json:"pem" msgpack:"pem"`
	// PrivateKey is the PEM private key of a stored certificate. The Core saves it as a
	// secret, and no endpoint returns it.
	PrivateKey string `json:"private_key" msgpack:"private_key"`
	// CertificatePath is the path of a file certificate on the computer that uses it.
	CertificatePath string `json:"certificate_path" msgpack:"certificate_path"`
	// Key selects the certificate to replace. A zero key creates a new certificate.
	Key Key `json:"key" msgpack:"key"`
}

func (in Input) certificate() (Certificate, error) {
	base := certificate.Base{Key: in.Key, Name: in.Name}
	v := validate.New("certificate")
	switch in.Source {
	case secret.SourceStored:
		const msg = "must be empty for a stored certificate"
		v.Ternary("certificate_path", in.CertificatePath != "", msg)
		v.Ternary("private_key_path", in.PrivateKeyPath != nil, msg)
		return Certificate{Variant: certificate.StoredCertificate{
			Base: base,
			Pem:  in.Pem,
		}}, v.Error()
	case secret.SourceFile:
		v.Ternary("pem", in.Pem != "", "must be empty for a file certificate")
		return Certificate{Variant: certificate.FileCertificate{
			Base:            base,
			CertificatePath: in.CertificatePath,
			PrivateKeyPath:  in.PrivateKeyPath,
			Details:         in.Details,
		}}, v.Error()
	}
	v.Ternaryf("source", true, "must be stored or file, not %q", in.Source)
	return Certificate{}, v.Error()
}

// CreateRequest holds the certificates to create or replace.
type CreateRequest struct {
	Certificates []Input `json:"certificates" msgpack:"certificates"`
}

// CreateResponse holds the created certificates.
type CreateResponse struct {
	Certificates []Certificate `json:"certificates" msgpack:"certificates"`
}

// Create creates the certificates, or replaces those whose key exists. It requires
// create access on certificates, and on secrets when a certificate carries a private
// key.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	objects := []ontology.ID{{Type: ontology.ResourceTypeCertificate}}
	if lo.ContainsBy(
		req.Certificates,
		func(in Input) bool { return in.PrivateKey != "" },
	) {
		objects = append(objects, ontology.ID{Type: ontology.ResourceTypeSecret})
	}
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: objects,
	}); err != nil {
		return CreateResponse{}, err
	}
	w := s.internal.NewWriter(tx)
	res := CreateResponse{Certificates: make([]Certificate, len(req.Certificates))}
	for i, in := range req.Certificates {
		c, err := in.certificate()
		if err != nil {
			return CreateResponse{}, err
		}
		if err = w.Create(ctx, &c, []byte(in.PrivateKey)); err != nil {
			return CreateResponse{}, err
		}
		res.Certificates[i] = c
	}
	return res, nil
}

// RetrieveRequest selects the certificates to retrieve.
type RetrieveRequest struct {
	// SearchTerm filters the certificates by name.
	SearchTerm string `json:"search_term" msgpack:"search_term"`
	// Keys selects certificates by key.
	Keys []Key `json:"keys" msgpack:"keys"`
	// Limit is the maximum number of certificates to return.
	Limit int `json:"limit" msgpack:"limit"`
	// Offset is the number of certificates to skip.
	Offset int `json:"offset" msgpack:"offset"`
}

// RetrieveResponse holds the retrieved certificates.
type RetrieveResponse struct {
	Certificates []Certificate `json:"certificates" msgpack:"certificates"`
}

// Retrieve returns the certificates the request selects. It requires retrieve access on
// each of them.
func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	q := s.internal.NewRetrieve().Search(req.SearchTerm)
	if req.Limit != 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset != 0 {
		q = q.Offset(req.Offset)
	}
	if len(req.Keys) != 0 {
		q = q.Where(certificate.MatchKeys(req.Keys...))
	}
	var certs []Certificate
	if err := q.Entries(&certs).Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: certificate.OntologyIDsFromCertificates(certs),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{Certificates: certs}, nil
}

// RenameRequest names the certificate to rename and its new name.
type RenameRequest struct {
	// Name is the new name. It must not be empty.
	Name string `json:"name" msgpack:"name"`
	// Key selects the certificate to rename.
	Key Key `json:"key" msgpack:"key"`
}

// Rename renames the certificate. It requires update access on it.
func (s *Service) Rename(
	ctx context.Context,
	tx gorp.Tx,
	req RenameRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{certificate.OntologyID(req.Key)},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
}

// DeleteRequest selects the certificates to delete.
type DeleteRequest struct {
	Keys []Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the certificates and the secrets that hold their private keys. It
// requires delete access on each certificate.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: certificate.OntologyIDs(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}
