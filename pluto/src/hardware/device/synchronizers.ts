// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device } from "@synnaxlabs/client";

import { Flux } from "@/flux";

export const useSetSynchronizer = (onSet: (device: device.Device) => void): void =>
  Flux.useListener({
    channel: device.SET_CHANNEL_NAME,
    onChange: Flux.parsedHandler(device.deviceZ, (args) => onSet(args.changed)),
  });

export const useDeleteSynchronizer = (onDelete: (key: device.Key) => void): void =>
  Flux.useListener({
    channel: device.DELETE_CHANNEL_NAME,
    onChange: Flux.parsedHandler(device.keyZ, (args) => onDelete(args.changed)),
  });

export const useStatusSynchronizer = (
  onStatusChange: (status: device.Status) => void,
): void =>
  Flux.useListener({
    channel: device.STATUS_CHANNEL_NAME,
    onChange: Flux.parsedHandler(device.statusZ, (args) =>
      onStatusChange(args.changed),
    ),
  });
