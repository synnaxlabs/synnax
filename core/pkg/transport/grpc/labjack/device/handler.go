// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device

import (
	"context"
	"strconv"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/labjack/device"
	"github.com/synnaxlabs/synnax/pkg/api/labjack/device/pb"
	svcdevice "github.com/synnaxlabs/synnax/pkg/service/labjack/device"
	devicepb "github.com/synnaxlabs/synnax/pkg/service/labjack/device/pb"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createServer = grpc.UnaryServer[
		device.CreateRequest,
		*CreateRequest,
		device.CreateResponse,
		*CreateResponse,
	]
	retrieveServer = grpc.UnaryServer[
		device.RetrieveRequest,
		*pb.RetrieveRequest,
		device.RetrieveResponse,
		*pb.RetrieveResponse,
	]
	deleteServer = grpc.UnaryServer[
		device.DeleteRequest,
		*DeleteRequest,
		struct{},
		*emptypb.Empty,
	]
	dispatchServer = grpc.UnaryServer[
		device.DispatchRequest,
		*DispatchRequest,
		struct{},
		*emptypb.Empty,
	]
)

type (
	createRequestTranslator    struct{}
	createResponseTranslator   struct{}
	retrieveRequestTranslator  struct{}
	retrieveResponseTranslator struct{}
	deleteRequestTranslator    struct{}
	dispatchRequestTranslator  struct{}
)

var (
	_ grpc.BackwardTranslator[device.CreateRequest, *CreateRequest]         = createRequestTranslator{}
	_ grpc.ForwardTranslator[device.CreateResponse, *CreateResponse]        = createResponseTranslator{}
	_ grpc.BackwardTranslator[device.RetrieveRequest, *pb.RetrieveRequest]  = retrieveRequestTranslator{}
	_ grpc.ForwardTranslator[device.RetrieveResponse, *pb.RetrieveResponse] = retrieveResponseTranslator{}
	_ grpc.BackwardTranslator[device.DeleteRequest, *DeleteRequest]         = deleteRequestTranslator{}
	_ grpc.BackwardTranslator[device.DispatchRequest, *DispatchRequest]     = dispatchRequestTranslator{}
)

func (createRequestTranslator) Backward(
	_ context.Context,
	req *CreateRequest,
) (device.CreateRequest, error) {
	devices, err := devicepb.DevicesFromPB(req.Devices)
	if err != nil {
		return device.CreateRequest{}, err
	}
	return device.CreateRequest{Devices: devices}, nil
}

func (createResponseTranslator) Forward(
	_ context.Context,
	res device.CreateResponse,
) (*CreateResponse, error) {
	devices, err := devicepb.DevicesToPB(res.Devices)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{Devices: devices}, nil
}

func (retrieveRequestTranslator) Backward(
	_ context.Context,
	req *pb.RetrieveRequest,
) (device.RetrieveRequest, error) {
	return pb.RetrieveRequestFromPB(req)
}

func (retrieveResponseTranslator) Forward(
	_ context.Context,
	res device.RetrieveResponse,
) (*pb.RetrieveResponse, error) {
	return pb.RetrieveResponseToPB(res)
}

func (deleteRequestTranslator) Backward(
	_ context.Context,
	req *DeleteRequest,
) (device.DeleteRequest, error) {
	keys, err := lo.MapErr(req.Keys, func(k string, _ int) (svcdevice.Key, error) {
		return uuid.Parse(k)
	})
	if err != nil {
		return device.DeleteRequest{}, err
	}
	return device.DeleteRequest{Keys: keys}, nil
}

func (dispatchRequestTranslator) Backward(
	_ context.Context,
	req *DispatchRequest,
) (device.DispatchRequest, error) {
	key, err := uuid.Parse(req.Key)
	if err != nil {
		return device.DispatchRequest{}, err
	}
	acts := make([]svcdevice.Action, len(req.Actions))
	for i, b := range req.Actions {
		if err = json.Codec.Decode(b, &acts[i]); err != nil {
			return device.DispatchRequest{}, validate.PathedError(
				errors.Wrap(validate.ErrValidation, err.Error()),
				"actions",
				strconv.Itoa(i),
			)
		}
	}
	return device.DispatchRequest{
		Key:         key,
		DispatchKey: req.DispatchKey,
		Actions:     acts,
	}, nil
}

// New binds the LabJack device routes to t and returns their gRPC transport.
func New(t *api.Transport, errs *errors.Codec) grpc.BindableTransport {
	create := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &LabJackDeviceCreateService_ServiceDesc,
		Errors:             errs,
	}
	t.LabJackDeviceCreate = create
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &LabJackDeviceRetrieveService_ServiceDesc,
		Errors:             errs,
	}
	t.LabJackDeviceRetrieve = retrieve
	del := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &LabJackDeviceDeleteService_ServiceDesc,
		Errors:             errs,
	}
	t.LabJackDeviceDelete = del
	dispatch := &dispatchServer{
		RequestTranslator:  dispatchRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &LabJackDeviceDispatchService_ServiceDesc,
		Errors:             errs,
	}
	t.LabJackDeviceDispatch = dispatch
	return grpc.CompoundBindableTransport{create, retrieve, del, dispatch}
}
