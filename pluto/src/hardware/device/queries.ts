// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, type Synnax } from "@synnaxlabs/client";
import { primitive, type record } from "@synnaxlabs/x";
import { useEffect } from "react";

import { Flux } from "@/flux";

export const FLUX_STORE_KEY = "devices";

export interface FluxStore extends Flux.UnaryStore<string, device.Device> {}

interface SubStore extends Flux.Store {
  devices: Flux.UnaryStore<device.Key, device.Device>;
}

const SET_DEVICE_LISTENER: Flux.ChannelListener<SubStore, typeof device.deviceZ> = {
  channel: device.SET_CHANNEL_NAME,
  schema: device.deviceZ,
  onChange: ({ store, changed }) =>
    store.devices.set(changed.key, (p) => {
      if (p == null) return changed;
      return { ...changed, status: p.status };
    }),
};

const DELETE_DEVICE_LISTENER: Flux.ChannelListener<SubStore, typeof device.keyZ> = {
  channel: device.DELETE_CHANNEL_NAME,
  schema: device.keyZ,
  onChange: ({ store, changed }) => store.devices.delete(changed),
};

const SET_STATUS_LISTENER: Flux.ChannelListener<SubStore, typeof device.statusZ> = {
  channel: device.STATUS_CHANNEL_NAME,
  schema: device.statusZ,
  onChange: ({ store, changed }) => {
    store.devices.set(changed.details.device, (p) =>
      p == null ? p : { ...p, status: changed },
    );
  },
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<SubStore> = {
  listeners: [SET_DEVICE_LISTENER, DELETE_DEVICE_LISTENER, SET_STATUS_LISTENER],
};

export const useSetSynchronizer = (onSet: (device: device.Device) => void): void => {
  const store = Flux.useStore<SubStore>();
  useEffect(() => store.devices.onSet(onSet), [store]);
};

const retrieveByKey = async <
  Properties extends record.Unknown = record.Unknown,
  Make extends string = string,
  Model extends string = string,
>(
  client: Synnax,
  store: SubStore,
  params: device.SingleRetrieveArgs,
): Promise<device.Device<Properties, Make, Model>> => {
  const cached = store.devices.get(params.key);
  if (cached != null) return cached as device.Device<Properties, Make, Model>;
  const device = await client.hardware.devices.retrieve<Properties, Make, Model>({
    ...params,
    includeStatus: true,
  });
  store.devices.set(params.key, device);
  return device;
};

export const createRetrieve = <
  Properties extends record.Unknown = record.Unknown,
  Make extends string = string,
  Model extends string = string,
>() =>
  Flux.createRetrieve<
    device.SingleRetrieveArgs,
    device.Device<Properties, Make, Model>,
    SubStore
  >({
    name: "Device",
    retrieve: async ({ client, params, store }) =>
      await retrieveByKey<Properties, Make, Model>(client, store, params),
    mountListeners: ({ store, onChange, params: { key } }) => [
      store.devices.onSet(
        (changed) => onChange(changed as device.Device<Properties, Make, Model>),
        key,
      ),
    ],
  });

export const {
  useRetrieve,
  useRetrieveStateful: useStatefulRetrieve,
  useRetrieveEffect,
} = createRetrieve();

export interface ListParams extends device.MultiRetrieveArgs {}

export const useList = Flux.createList<ListParams, device.Key, device.Device, SubStore>(
  {
    name: "Devices",
    retrieveCached: ({ store, params }) =>
      store.devices.get((d) => {
        if (primitive.isNonZero(params.makes) && !params.makes.includes(d.make))
          return false;
        if (primitive.isNonZero(params.models) && !params.models.includes(d.model))
          return false;
        if (primitive.isNonZero(params.racks) && !params.racks.includes(d.rack))
          return false;
        if (
          primitive.isNonZero(params.locations) &&
          !params.locations.includes(d.location)
        )
          return false;
        if (primitive.isNonZero(params.names) && !params.names.includes(d.name))
          return false;
        if (primitive.isNonZero(params.keys) && !params.keys.includes(d.key))
          return false;
        return true;
      }),
    retrieve: async ({ client, params, store }) => {
      const devices = await client.hardware.devices.retrieve({
        includeStatus: true,
        ...params,
      });
      devices.forEach((d) => store.devices.set(d.key, d));
      return devices;
    },
    retrieveByKey: async ({ client, key, store }) =>
      await retrieveByKey(client, store, { key }),
    mountListeners: ({ store, onChange, onDelete }) => [
      store.devices.onSet((changed) => onChange(changed.key, changed)),
      store.devices.onDelete(onDelete),
    ],
  },
);
