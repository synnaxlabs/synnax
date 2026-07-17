// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, NotFoundError, ontology } from "@synnaxlabs/client";
import { array, errors, primitive, type record, uuid } from "@synnaxlabs/x";
import { useEffect } from "react";
import { type z } from "zod";

import { Flux } from "@/flux";
import { Ontology } from "@/ontology";
import { type Rack } from "@/rack";
import { state } from "@/state";
import { Status } from "@/status";
import { type Task } from "@/task";

export const FLUX_STORE_KEY = "devices";
const RESOURCE_NAME = "device";
const PLURAL_RESOURCE_NAME = "devices";

// Explicitly omit 'status' from the device type to make sure we aren't storing two
// copies of the statuses in the flux store.
export interface FluxStore extends Flux.UnaryStore<
  string,
  Omit<device.Device, "status">
> {}

export interface FluxSubStore extends Task.FluxSubStore {
  [FLUX_STORE_KEY]: FluxStore;
  [Rack.FLUX_STORE_KEY]: Rack.FluxStore;
  [Status.FLUX_STORE_KEY]: Status.FluxStore;
}

const genericDeviceZ = device.deviceZ();

const SET_DEVICE_LISTENER: Flux.ChannelListener<FluxSubStore, typeof genericDeviceZ> = {
  channel: device.SET_CHANNEL_NAME,
  schema: genericDeviceZ,
  onChange: ({ store, changed }) => store.devices.set(changed.key, changed),
};

const DELETE_DEVICE_LISTENER: Flux.ChannelListener<FluxSubStore, typeof device.keyZ> = {
  channel: device.DELETE_CHANNEL_NAME,
  schema: device.keyZ,
  onChange: ({ store, changed }) => store.devices.delete(changed),
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<FluxSubStore> = {
  listeners: [SET_DEVICE_LISTENER, DELETE_DEVICE_LISTENER],
};

export const useSetSynchronizer = (onSet: (device: device.Device) => void): void => {
  const store = Flux.useStore<FluxSubStore>();
  useEffect(() => store.devices.onSet(onSet), [store]);
};

export type RetrieveQuery = device.RetrieveSingleParams;

const BASE_QUERY: Partial<RetrieveQuery> = { includeStatus: true };

export const retrieveSingle = async <
  Properties extends z.ZodType<record.Unknown> = z.ZodType<record.Unknown>,
  Make extends z.ZodType<string> = z.ZodString,
  Model extends z.ZodType<string> = z.ZodString,
>({
  client,
  store,
  query,
  schemas,
}: Flux.RetrieveParams<RetrieveQuery, FluxSubStore> & {
  schemas?: device.DeviceSchemas<Properties, Make, Model>;
}): Promise<device.Device<Properties, Make, Model>> => {
  const cached = store.devices.get(query.key);
  if (cached != null) {
    // The cache is also fed by generic writers (list retrieves, streamed set
    // events) that never apply vendor schemas, so a cached entry may predate the
    // vendor's migrations and defaults. Trust it only if it satisfies the
    // requested schemas; otherwise fall through to a network retrieve, which
    // parses and overwrites the cached entry.
    const parsed =
      schemas == null ? cached : device.deviceZ(schemas).safeParse(cached).data;
    if (parsed != null) {
      let status: device.Status | undefined;
      try {
        status = await Status.retrieveSingle({
          store,
          client,
          query: { key: device.statusKey(query.key) },
          detailsSchema: device.statusDetailsZ,
        });
      } catch (err) {
        if (!NotFoundError.matches(err)) throw errors.fromUnknown(err);
      }
      return { ...parsed, status } as device.Device<Properties, Make, Model>;
    }
  }
  const dev =
    schemas != null
      ? await client.devices.retrieve({ ...BASE_QUERY, ...query, schemas })
      : await client.devices.retrieve({ ...BASE_QUERY, ...query });
  store.devices.set(dev.key, dev);
  if (dev.status != null) store.statuses.set(dev.status);
  return dev as device.Device<Properties, Make, Model>;
};

export type RetrieveMultipleQuery = {
  keys: device.Key[];
};

export const retrieveMultiple = async <
  Properties extends z.ZodType<record.Unknown> = z.ZodType<record.Unknown>,
  Make extends z.ZodType<string> = z.ZodString,
  Model extends z.ZodType<string> = z.ZodString,
>({
  client,
  store,
  query: { keys },
  schemas,
}: Flux.RetrieveParams<RetrieveMultipleQuery, FluxSubStore> & {
  schemas?: device.DeviceSchemas<Properties, Make, Model>;
}): Promise<device.Device<Properties, Make, Model>[]> => {
  const cached = store.devices.get(keys);
  const cachedKeys = new Set(cached.map((d) => d.key));
  const missingKeys = keys.filter((key) => !cachedKeys.has(key));

  const statusKeys = cached.map((d) => device.statusKey(d.key));
  const statuses = await Status.retrieveMultiple({
    store,
    client,
    query: { keys: statusKeys },
  });
  const statusMap = new Map(statuses.map((s) => [s.key, s]));
  const cachedWithStatus = cached.map((d) => {
    const status = statusMap.get(device.statusKey(d.key));
    return { ...d, status } as device.Device<Properties, Make, Model>;
  });

  const devices = [...cachedWithStatus];
  if (missingKeys.length > 0) {
    const fetched =
      schemas != null
        ? await client.devices.retrieve({ ...BASE_QUERY, keys: missingKeys, schemas })
        : await client.devices.retrieve({ ...BASE_QUERY, keys: missingKeys });
    devices.push(...(fetched as device.Device<Properties, Make, Model>[]));
    store.devices.set(fetched);
    fetched.forEach((d) => {
      if (d.status != null) store.statuses.set(d.status);
    });
  }

  return Flux.orderByKeys(keys, devices, (d) => d.key);
};

export const createRetrieve = <
  Properties extends z.ZodType<record.Unknown> = z.ZodType<record.Unknown>,
  Make extends z.ZodType<string> = z.ZodString,
  Model extends z.ZodType<string> = z.ZodString,
>(
  schemas?: device.DeviceSchemas<Properties, Make, Model>,
) =>
  Flux.createRetrieve<
    RetrieveQuery,
    device.Device<Properties, Make, Model>,
    FluxSubStore
  >({
    name: RESOURCE_NAME,
    retrieve: (params) =>
      retrieveSingle<Properties, Make, Model>({ ...params, schemas }),
    mountListeners: ({ store, onChange, query: { key } }) => [
      store.devices.onSet(
        (changed) =>
          onChange((p) => ({
            ...p,
            ...(changed as device.Device<Properties, Make, Model>),
            status: p?.status,
          })),
        key,
      ),
      store.statuses.onSet((status) => {
        const parsed = device.statusZ.parse(status);
        onChange(state.skipUndefined((p) => ({ ...p, status: parsed })));
      }, device.statusKey(key)),
    ],
  });

export const {
  useRetrieve,
  useRetrieveStateful: useStatefulRetrieve,
  useRetrieveEffect,
} = createRetrieve();

export type ListParams = device.RetrieveMultipleParams;

export const useList = Flux.createList<
  ListParams,
  device.Key,
  device.Device,
  FluxSubStore
>({
  name: PLURAL_RESOURCE_NAME,
  retrieveCached: ({ store, query: params }) =>
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
  retrieve: async ({ client, query, store }) => {
    const devices = await client.devices.retrieve({ ...BASE_QUERY, ...query });
    store.devices.set(devices);
    return devices;
  },
  retrieveByKey: async ({ key, ...rest }) =>
    await retrieveSingle({ ...rest, query: { key } }),
  mountListeners: ({ store, onChange, onDelete }) => [
    store.devices.onSet((changed) => onChange(changed.key, changed)),
    store.statuses.onSet((status) => {
      const parsed = device.statusZ.safeParse(status);
      if (!parsed.success) return;
      onChange(
        parsed.data.details.device,
        state.skipNull((p) => ({ ...p, status: parsed.data })),
      );
    }),
    store.devices.onDelete(onDelete),
  ],
});

export type UseDeleteArgs = device.Key | device.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteArgs, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, store, rollbacks, onOptimisticComplete }) => {
    const keys = array.toArray(data);
    const ids = device.ontologyID(keys);
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.push(store.relationships.delete(relFilter));
    rollbacks.push(store.resources.delete(ontology.idToString(ids)));
    rollbacks.push(store.devices.delete(keys));
    await onOptimisticComplete(data);
    await client.devices.delete(keys);
    return data;
  },
});

export const createCreate = <
  Properties extends z.ZodType<record.Unknown> = z.ZodType<record.Unknown>,
  Make extends z.ZodType<string> = z.ZodString,
  Model extends z.ZodType<string> = z.ZodString,
>(
  schemas?: device.DeviceSchemas<Properties, Make, Model>,
) =>
  Flux.createUpdate<
    device.New<Properties, Make, Model>,
    FluxSubStore,
    device.Device<Properties, Make, Model>
  >({
    name: RESOURCE_NAME,
    verbs: Flux.CREATE_VERBS,
    update: async ({ data, client, rollbacks, store }) => {
      const dev =
        schemas != null
          ? await client.devices.create(data, schemas)
          : await client.devices.create(data as device.New);
      rollbacks.push(store.devices.set(dev));
      return dev as device.Device<Properties, Make, Model>;
    },
  });

export const { useUpdate: useCreate } = createCreate();

export type UseRetrieveGroupArgs = Record<string, never>;

export const { useRetrieve: useRetrieveGroupID } = Flux.createRetrieve<
  UseRetrieveGroupArgs,
  ontology.ID | undefined,
  FluxSubStore
>({
  name: "Device Group",
  retrieve: async ({ client, store }) => {
    const children = await client.ontology.retrieveChildren(ontology.ROOT_ID);
    store.resources.set(children);
    const groupChildren = children.filter((r) => r.id.type === "group");
    if (groupChildren.length === 0) return undefined;
    const groups = await client.groups.retrieve({
      keys: groupChildren.map((r) => r.id.key),
    });
    const devicesGroup = groups.find((g) => g.name === "Devices");
    return groupChildren.find((r) => r.id.key === devicesGroup?.key)?.id;
  },
});

export interface RenameParams extends Pick<device.Device, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ data, client, rollbacks, store, onOptimisticComplete }) => {
    const { key, name } = data;
    const dev = await retrieveSingle({ client, store, query: { key } });
    const renamed = { ...dev, name };
    rollbacks.push(store.devices.set(renamed));
    await onOptimisticComplete(data);
    await client.devices.create(renamed);
    return data;
  },
});

export const formSchema = device.deviceZ();

export type FormQuery = RetrieveQuery;

export const createForm = <
  Properties extends z.ZodType<record.Unknown> = z.ZodType<record.Unknown>,
  Make extends z.ZodType<string> = z.ZodString,
  Model extends z.ZodType<string> = z.ZodString,
>(
  schemas?: device.DeviceSchemas<Properties, Make, Model>,
) =>
  Flux.createForm<FormQuery, typeof formSchema, FluxSubStore>({
    name: RESOURCE_NAME,
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
    retrieve: async ({ query, client, reset, store, set }) => {
      if (primitive.isZero(query.key)) {
        set("key", uuid.create());
        return;
      }
      const dev = await retrieveSingle({ client, store, query, schemas });
      reset(dev);
    },
    update: async ({ value, client, store, rollbacks }) => {
      const data = value();
      const result =
        schemas != null
          ? await client.devices.create(
              data as device.New<Properties, Make, Model>,
              schemas,
            )
          : await client.devices.create(data);
      rollbacks.push(store.devices.set(result.key, result));
    },
    mountListeners: ({ store, query: { key }, reset, set }) => {
      if (primitive.isZero(key)) return [];
      const schema = device.deviceZ(schemas);
      return [
        // Streamed set events are parsed generically, so they can carry shapes
        // that predate the vendor's migrations and defaults. Only reset the form
        // when the event satisfies the vendor schemas.
        store.devices.onSet((changed) => {
          const parsed = schema.safeParse(changed);
          if (parsed.success) reset(parsed.data as z.infer<typeof formSchema>);
        }, key),
        store.statuses.onSet(
          (status) => set("status", device.statusZ.parse(status)),
          device.statusKey(key),
        ),
      ];
    },
  });

export const useForm = createForm();
