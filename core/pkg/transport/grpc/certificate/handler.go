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
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/certificate"
	"github.com/synnaxlabs/synnax/pkg/api/certificate/pb"
	certificatepb "github.com/synnaxlabs/synnax/pkg/service/certificate/pb"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/x/errors"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	retrieveServer = grpc.UnaryServer[
		certificate.RetrieveRequest,
		*pb.RetrieveRequest,
		certificate.RetrieveResponse,
		*pb.RetrieveResponse,
	]
	setServer = grpc.UnaryServer[
		certificate.SetRequest,
		*SetRequest,
		certificate.SetResponse,
		*SetResponse,
	]
	refreshServer = grpc.UnaryServer[
		certificate.RefreshRequest,
		*RefreshRequest,
		struct{},
		*emptypb.Empty,
	]
)

type (
	retrieveRequestTranslator  struct{}
	retrieveResponseTranslator struct{}
	setRequestTranslator       struct{}
	setResponseTranslator      struct{}
	refreshRequestTranslator   struct{}
)

var (
	_ grpc.BackwardTranslator[certificate.RetrieveRequest, *pb.RetrieveRequest]  = retrieveRequestTranslator{}
	_ grpc.ForwardTranslator[certificate.RetrieveResponse, *pb.RetrieveResponse] = retrieveResponseTranslator{}
	_ grpc.BackwardTranslator[certificate.SetRequest, *SetRequest]               = setRequestTranslator{}
	_ grpc.ForwardTranslator[certificate.SetResponse, *SetResponse]              = setResponseTranslator{}
	_ grpc.BackwardTranslator[certificate.RefreshRequest, *RefreshRequest]       = refreshRequestTranslator{}
)

func (retrieveRequestTranslator) Backward(
	_ context.Context,
	req *pb.RetrieveRequest,
) (certificate.RetrieveRequest, error) {
	return pb.RetrieveRequestFromPB(req)
}

func (retrieveResponseTranslator) Forward(
	_ context.Context,
	res certificate.RetrieveResponse,
) (*pb.RetrieveResponse, error) {
	return pb.RetrieveResponseToPB(res)
}

func inputFromPB(in *Input) (certificate.Input, error) {
	out := certificate.Input{
		Name:            in.Name,
		Source:          secret.Source(in.Source),
		Pem:             in.Pem,
		PrivateKeyPem:   in.PrivateKeyPem,
		CertificatePath: in.CertificatePath,
		PrivateKeyFile:  in.PrivateKeyFile,
		Rack:            rack.Key(in.Rack),
	}
	if in.Key != "" {
		key, err := uuid.Parse(in.Key)
		if err != nil {
			return certificate.Input{}, err
		}
		out.Key = key
	}
	if in.Details != nil {
		details, err := certificatepb.DetailsFromPB(in.Details)
		if err != nil {
			return certificate.Input{}, err
		}
		out.Details = &details
	}
	return out, nil
}

func (setRequestTranslator) Backward(
	_ context.Context,
	req *SetRequest,
) (certificate.SetRequest, error) {
	inputs, err := lo.MapErr(
		req.Certificates,
		func(in *Input, _ int) (certificate.Input, error) { return inputFromPB(in) },
	)
	if err != nil {
		return certificate.SetRequest{}, err
	}
	return certificate.SetRequest{Certificates: inputs}, nil
}

func (setResponseTranslator) Forward(
	_ context.Context,
	res certificate.SetResponse,
) (*SetResponse, error) {
	certs, err := certificatepb.CertificatesToPB(res.Certificates)
	if err != nil {
		return nil, err
	}
	return &SetResponse{Certificates: certs}, nil
}

func (refreshRequestTranslator) Backward(
	_ context.Context,
	req *RefreshRequest,
) (certificate.RefreshRequest, error) {
	key, err := uuid.Parse(req.Key)
	if err != nil {
		return certificate.RefreshRequest{}, err
	}
	details, err := certificatepb.DetailsFromPB(req.Details)
	if err != nil {
		return certificate.RefreshRequest{}, err
	}
	return certificate.RefreshRequest{Key: key, Details: details}, nil
}

// New binds the certificate routes that the C++ client calls to t and returns their
// gRPC transport.
func New(t *api.Transport, errs *errors.Codec) grpc.BindableTransport {
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &CertificateRetrieveService_ServiceDesc,
		Errors:             errs,
	}
	t.CertificateRetrieve = retrieve
	set := &setServer{
		RequestTranslator:  setRequestTranslator{},
		ResponseTranslator: setResponseTranslator{},
		ServiceDesc:        &CertificateSetService_ServiceDesc,
		Errors:             errs,
	}
	t.CertificateSet = set
	refresh := &refreshServer{
		RequestTranslator:  refreshRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &CertificateRefreshService_ServiceDesc,
		Errors:             errs,
	}
	t.CertificateRefresh = refresh
	return grpc.CompoundBindableTransport{retrieve, set, refresh}
}
