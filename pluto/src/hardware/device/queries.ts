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
import { Sync } from "@/flux/sync";

export const useSetSynchronizer = (onSet: (device: device.Device) => void): void =>
  Sync.useListener({
    channel: device.SET_CHANNEL_NAME,
    onChange: Sync.parsedHandler(device.deviceZ, async (args) => {
      onSet(args.changed);
    }),
  });

export const useDeleteSynchronizer = (onDelete: (key: device.Key) => void): void =>
  Sync.useListener({
    channel: device.DELETE_CHANNEL_NAME,
    onChange: Sync.parsedHandler(device.keyZ, async (args) => {
      onDelete(args.changed);
    }),
  });

export const useStatusSynchronizer = (
  onStatusChange: (status: device.Status) => void,
): void =>
  Sync.useListener({
    channel: device.STATUS_CHANNEL_NAME,
    onChange: Sync.parsedHandler(device.statusZ, async (args) => {
      onStatusChange(args.changed);
    }),
  });

export interface ListParams extends Flux.Params {
  term?: string;
  offset?: number;
  limit?: number;
}

export const useList = Flux.createList<ListParams, device.Key, device.Device>({
  name: "Devices",
  retrieve: async ({ client, params }) =>
    await client.hardware.devices.search(params.term ?? ""),
  retrieveByKey: async ({ client, key }) => await client.hardware.devices.retrieve(key),
});
