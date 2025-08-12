// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device } from "@synnaxlabs/client";
import { type record } from "@synnaxlabs/x";

import { Flux } from "@/flux";

export const useSetSynchronizer = (onSet: (device: device.Device) => void): void =>
  Flux.useListener({
    channel: device.SET_CHANNEL_NAME,
    onChange: Flux.parsedHandler(device.deviceZ, async ({ changed }) => onSet(changed)),
  });

export interface RetrieveParams {
  key: device.Key;
}

export const retrieve = <
  Properties extends record.Unknown = record.Unknown,
  Make extends string = string,
  Model extends string = string,
>() =>
  Flux.createRetrieve<RetrieveParams, device.Device<Properties, Make, Model>>({
    name: "Device",
    retrieve: async ({ client, params }) =>
      await client.hardware.devices.retrieve({ key: params.key, includeStatus: true }),
    listeners: [
      {
        channel: device.SET_CHANNEL_NAME,
        onChange: Flux.parsedHandler(device.deviceZ, async ({ onChange, changed }) =>
          onChange((p) => {
            if (changed.key !== p.key) return p;
            return changed as device.Device<Properties, Make, Model>;
          }),
        ),
      },
      {
        channel: device.STATUS_CHANNEL_NAME,
        onChange: Flux.parsedHandler(device.statusZ, async ({ changed, onChange }) => {
          onChange((p) => {
            if (changed.details.device !== p.key) return p;
            p.status = changed;
            return p;
          });
        }),
      },
    ],
  });

export interface ListParams extends device.MultiRetrieveArgs {}

export const useList = Flux.createList<ListParams, device.Key, device.Device>({
  name: "Devices",
  retrieve: async ({ client, params }) =>
    await client.hardware.devices.retrieve({ includeStatus: true, ...params }),
  retrieveByKey: async ({ client, key }) =>
    await client.hardware.devices.retrieve({ key, includeStatus: true }),
  listeners: [
    {
      channel: device.SET_CHANNEL_NAME,
      onChange: Flux.parsedHandler(device.deviceZ, async ({ onChange, changed }) =>
        onChange(changed.key, changed),
      ),
    },
    {
      channel: device.DELETE_CHANNEL_NAME,
      onChange: Flux.parsedHandler(device.keyZ, async ({ changed, onDelete }) =>
        onDelete(changed),
      ),
    },
    {
      channel: device.STATUS_CHANNEL_NAME,
      onChange: Flux.parsedHandler(device.statusZ, async ({ changed, onChange }) => {
        onChange(changed.details.device, (p) =>
          p == null ? p : { ...p, status: changed },
        );
      }),
    },
  ],
});
