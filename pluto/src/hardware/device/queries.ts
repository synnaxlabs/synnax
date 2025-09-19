// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, ontology, type Synnax } from "@synnaxlabs/client";
import { array, primitive, type record, uuid } from "@synnaxlabs/x";
import { useEffect } from "react";

import { Flux } from "@/flux";
import { type Rack } from "@/hardware/rack";
import { type Task } from "@/hardware/task";
import { Ontology } from "@/ontology";

export const FLUX_STORE_KEY = "devices";

type ChangeVariant = "payload" | "status";

export interface FluxStore
  extends Flux.UnaryStore<string, device.Device, ChangeVariant> {}

export interface FluxSubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
  [Task.FLUX_STORE_KEY]: Task.FluxStore;
  [Rack.FLUX_STORE_KEY]: Rack.FluxStore;
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.ResourceFluxStore;
}

const SET_DEVICE_LISTENER: Flux.ChannelListener<FluxSubStore, typeof device.deviceZ> = {
  channel: device.SET_CHANNEL_NAME,
  schema: device.deviceZ,
  onChange: ({ store, changed }) =>
    store.devices.set(
      changed.key,
      (p) => (p == null ? changed : { ...changed, status: p.status }),
      "payload",
    ),
};

const DELETE_DEVICE_LISTENER: Flux.ChannelListener<FluxSubStore, typeof device.keyZ> = {
  channel: device.DELETE_CHANNEL_NAME,
  schema: device.keyZ,
  onChange: ({ store, changed }) => store.devices.delete(changed),
};

const SET_STATUS_LISTENER: Flux.ChannelListener<FluxSubStore, typeof device.statusZ> = {
  channel: device.STATUS_CHANNEL_NAME,
  schema: device.statusZ,
  onChange: ({ store, changed }) => {
    store.devices.set(
      changed.details.device,
      (p) => (p == null ? p : { ...p, status: changed }),
      "status",
    );
  },
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<FluxSubStore> = {
  listeners: [SET_DEVICE_LISTENER, DELETE_DEVICE_LISTENER, SET_STATUS_LISTENER],
};

export const useSetSynchronizer = (onSet: (device: device.Device) => void): void => {
  const store = Flux.useStore<FluxSubStore>();
  useEffect(() => store.devices.onSet(onSet), [store]);
};

const retrieveSingle = async <
  Properties extends record.Unknown = record.Unknown,
  Make extends string = string,
  Model extends string = string,
>(
  client: Synnax,
  store: FluxSubStore,
  params: UseRetrieveArgs,
): Promise<device.Device<Properties, Make, Model>> => {
  const cached = store.devices.get(params.key);
  if (cached != null) return cached as device.Device<Properties, Make, Model>;
  const device = await client.hardware.devices.retrieve<Properties, Make, Model>({
    ...params,
    includeStatus: true,
  });
  store.devices.set(device, "payload");
  return device;
};

export interface UseRetrieveArgs extends device.SingleRetrieveArgs {}

export const createRetrieve = <
  Properties extends record.Unknown = record.Unknown,
  Make extends string = string,
  Model extends string = string,
>() =>
  Flux.createRetrieve<
    UseRetrieveArgs,
    device.Device<Properties, Make, Model>,
    FluxSubStore
  >({
    name: "Device",
    retrieve: async ({ client, params, store }) =>
      await retrieveSingle<Properties, Make, Model>(client, store, params),
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

export const useList = Flux.createList<
  ListParams,
  device.Key,
  device.Device,
  FluxSubStore
>({
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
    devices.forEach((d) => store.devices.set(d, "payload"));
    return devices;
  },
  retrieveByKey: async ({ client, key, store }) =>
    await retrieveSingle(client, store, { key }),
  mountListeners: ({ store, onChange, onDelete }) => [
    store.devices.onSet((changed) => onChange(changed.key, changed)),
    store.devices.onDelete(onDelete),
  ],
});

export type UseDeleteArgs = device.Key | device.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteArgs, FluxSubStore>({
  name: "Device",
  update: async ({ client, value, store, rollbacks }) => {
    const keys = array.toArray(value);
    const ids = keys.map((key) => device.ontologyID(key));
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.add(store.relationships.delete(relFilter));
    rollbacks.add(store.resources.delete(ontology.idToString(ids)));
    rollbacks.add(store.devices.delete(keys));
    await client.hardware.devices.delete(keys);
    return value;
  },
});

export interface UseUpdateArgs extends device.New {}

export const { useUpdate: useCreate } = Flux.createUpdate<
  UseUpdateArgs,
  FluxSubStore,
  device.Device
>({
  name: "Device",
  update: async ({ value, client, rollbacks, store }) => {
    const dev = await client.hardware.devices.create(value);
    rollbacks.add(store.devices.set(dev, "payload"));
    return dev;
  },
});

export interface UseRetrieveGroupArgs {}

export const { useRetrieve: useRetrieveGroupID } = Flux.createRetrieve<
  UseRetrieveGroupArgs,
  ontology.ID | undefined,
  FluxSubStore
>({
  name: "Device Group",
  retrieve: async ({ client, store }) => {
    const rels = store.relationships.get((rel) =>
      ontology.matchRelationship(rel, {
        from: ontology.ROOT_ID,
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
      }),
    );
    const groups = store.resources.get(rels.map((rel) => ontology.idToString(rel.to)));
    const cachedRes = groups.find((group) => group.name === "Devices");
    if (cachedRes != null) return cachedRes.id;
    const res = await client.ontology.retrieveChildren(ontology.ROOT_ID);
    store.resources.set(res);
    return res.find((r) => r.name === "Devices")?.id;
  },
});

export interface UseRenameArgs {
  key: device.Key;
  name: string;
}

export const { useUpdate: useRename } = Flux.createUpdate<UseRenameArgs, FluxSubStore>({
  name: "Device",
  update: async ({ value, client, rollbacks, store }) => {
    const { key, name } = value;
    const dev = await retrieveSingle(client, store, { key });
    rollbacks.add(store.devices.set(dev, "payload"));
    await client.hardware.devices.create({ ...dev, name });
    return value;
  },
});

export const formSchema = device.deviceZ;

const getInitialRackKey = async (client: Synnax, store: FluxSubStore) => {
  let rack = store.racks.get(() => true)[0];
  if (rack != null) return rack.key;
  rack = (
    await client.hardware.racks.retrieve({
      offset: 0,
      limit: 1,
    })
  )[0];
  return rack?.key ?? 0;
};

export const createForm = <
  Properties extends record.Unknown = record.Unknown,
  Make extends string = string,
  Model extends string = string,
>() =>
  Flux.createForm<UseRetrieveArgs, typeof formSchema, FluxSubStore>({
    name: "Device",
    schema: formSchema,
    initialValues: {
      key: "",
      rack: 0,
      name: "",
      make: "",
      model: "",
      location: "",
      configured: true,
      properties: {},
    },
    retrieve: async ({ params, client, reset, store, set }) => {
      if (primitive.isZero(params.key)) {
        set("rack", await getInitialRackKey(client, store));
        set("key", uuid.create());
        return;
      }
      const device = await retrieveSingle<Properties, Make, Model>(
        client,
        store,
        params,
      );
      reset(device);
    },
    update: async ({ value, client, store, rollbacks }) => {
      const result = await client.hardware.devices.create(value());
      rollbacks.add(store.devices.set(result, "payload"));
    },
    mountListeners: ({ store, params, reset }) => {
      const { key } = params;
      if (primitive.isZero(key)) return [];
      return store.devices.onSet((device, variant) => {
        if (variant === "payload") reset(device);
      }, key);
    },
  });

export const useForm = createForm();
