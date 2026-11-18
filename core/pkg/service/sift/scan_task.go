// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sift

import (
	"context"
	"uuid"

	assetsv1 "github.com/sift-stack/sift/go/gen/sift/assets/v1"
	pingv1 "github.com/sift-stack/sift/go/gen/sift/ping/v1"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/secret/grant"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	// testConnectionCommand pings a proposed Sift connection.
	testConnectionCommand = "test_connection"
	// listAssetsCommand lists the assets that a Sift device can see.
	listAssetsCommand = "list_assets"
	// uploadCommand queues an upload of the data of a range or a time range.
	uploadCommand = "upload"
	// listUploadsCommand lists the uploads of a range.
	listUploadsCommand = "list_uploads"
)

type testConnectionArgs struct {
	// Connection is the connection to test.
	Connection Properties `json:"connection"`
	// Grant lets the rack of the task read the API key of the connection.
	Grant grant.Key `json:"grant"`
}

type listAssetsArgs struct {
	// Device is the key of the Sift device to list the assets of.
	Device device.Key `json:"device"`
}

type listAssetsResult struct {
	// Assets are the names of the assets, in the order Sift lists them.
	Assets []string `json:"assets"`
	// Last is the asset of the newest upload from the device, or empty.
	Last string `json:"last"`
}

type listUploadsArgs struct {
	// Range is the key of the range to list the uploads of.
	Range ranger.Key `json:"range"`
}

// validate rejects a zero range, which names no range.
func (a listUploadsArgs) validate() error {
	v := validate.New("sift.list_uploads")
	v.Ternary("range", a.Range == uuid.Nil(), "required")
	return v.Error()
}

type listUploadsResult struct {
	// Uploads are the keys of the statuses of the uploads, newest first.
	Uploads []status.Key `json:"uploads"`
}

// listAssetsPageSize is the most assets Sift returns in one page.
const listAssetsPageSize = 1000

// scanner checks Sift devices and answers their commands.
type scanner struct {
	cfg FactoryConfig
	// rack is the rack of the task, which reads the API key a connection test names.
	rack rack.Key
}

var _ driver.Scanner = scanner{}

// Check implements driver.Scanner.
func (s scanner) Check(ctx context.Context, dev device.Device) error {
	return s.withDevice(
		ctx,
		dev,
		func(ctx context.Context, conn *grpc.ClientConn) error {
			_, err := pingv1.NewPingServiceClient(conn).Ping(ctx, &pingv1.PingRequest{})
			return err
		},
	)
}

// Exec implements driver.Scanner.
func (s scanner) Exec(
	ctx context.Context,
	cmd task.Command,
) (msgpack.EncodedJSON, error) {
	switch cmd.Type {
	case testConnectionCommand:
		return nil, s.testConnection(ctx, cmd)
	case listAssetsCommand:
		return s.listAssets(ctx, cmd)
	case uploadCommand:
		key, err := queueUpload(ctx, s.cfg, s.rack, cmd.Args)
		if err != nil {
			return nil, driver.Failure{
				Message:     "Failed to upload to Sift",
				Description: err.Error(),
			}
		}
		return msgpack.NewEncodedJSON(uploadResult{Status: key})
	case listUploadsCommand:
		var args listUploadsArgs
		if err := cmd.Args.Unmarshal(&args); err != nil {
			return nil, errors.Wrap(err, "invalid arguments")
		}
		if err := args.validate(); err != nil {
			return nil, err
		}
		keys, err := rangeUploads(ctx, s.cfg.DB, s.cfg.Status, args.Range)
		if err != nil {
			return nil, err
		}
		return msgpack.NewEncodedJSON(listUploadsResult{Uploads: keys})
	default:
		return nil, driver.ErrUnsupportedCommand
	}
}

func (s scanner) testConnection(ctx context.Context, cmd task.Command) error {
	var args testConnectionArgs
	if err := cmd.Args.Unmarshal(&args); err != nil {
		return errors.Wrap(err, "invalid connection")
	}
	if err := args.Connection.validate(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, s.cfg.Timeout.Duration())
	defer cancel()
	values, err := s.cfg.Secret.Resolve(
		ctx,
		s.rack.ResourceID(),
		args.Grant,
		args.Connection.APIKey,
	)
	if err != nil {
		return err
	}
	conn, err := dial(args.Connection.URI, string(values[args.Connection.APIKey]))
	if err != nil {
		return err
	}
	_, err = pingv1.NewPingServiceClient(conn).Ping(ctx, &pingv1.PingRequest{})
	return errors.Combine(failure(args.Connection.URI, err), conn.Close())
}

func (s scanner) listAssets(
	ctx context.Context,
	cmd task.Command,
) (msgpack.EncodedJSON, error) {
	var args listAssetsArgs
	if err := cmd.Args.Unmarshal(&args); err != nil {
		return nil, errors.Wrap(err, "invalid arguments")
	}
	var dev device.Device
	if err := s.cfg.Device.NewRetrieve().
		Where(device.MatchKeys(args.Device)).
		Entry(&dev).
		Exec(ctx, s.cfg.DB); err != nil {
		return nil, err
	}
	res := listAssetsResult{Assets: []string{}}
	if err := s.withDevice(
		ctx,
		dev,
		func(ctx context.Context, conn *grpc.ClientConn) error {
			client := assetsv1.NewAssetServiceClient(conn)
			req := &assetsv1.ListAssetsRequest{PageSize: listAssetsPageSize}
			for {
				page, err := client.ListAssets(ctx, req)
				if err != nil {
					return err
				}
				for _, a := range page.Assets {
					res.Assets = append(res.Assets, a.Name)
				}
				if page.NextPageToken == "" {
					return nil
				}
				req.PageToken = page.NextPageToken
			}
		},
	); err != nil {
		return nil, err
	}
	var err error
	if res.Last, err = lastAsset(ctx, s.cfg.DB, s.cfg.Status, dev.Key); err != nil {
		return nil, err
	}
	return msgpack.NewEncodedJSON(res)
}

// withDevice connects to the Sift API of dev and calls f with the connection. f must
// return within the factory timeout.
func (s scanner) withDevice(
	ctx context.Context,
	dev device.Device,
	f func(context.Context, *grpc.ClientConn) error,
) error {
	props, err := decodeProperties(dev)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, s.cfg.Timeout.Duration())
	defer cancel()
	conn, err := connect(ctx, s.cfg.Secret, s.cfg.DB, props)
	if err != nil {
		return err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			s.cfg.L.Warn("failed to close connection", zap.Error(err))
		}
	}()
	return failure(props.URI, f(ctx, conn))
}
