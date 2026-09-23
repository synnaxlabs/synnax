// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, ontology, query as clientQuery, type query } from "@synnaxlabs/client";
import { primitive, type record, uuid, verbs, zod } from "@synnaxlabs/x";
import { useEffect } from "react";
import { type z } from "zod";

import { Flux } from "@/flux";
import { Synnax } from "@/synnax";

const RESOURCE_NAME = "device";
const PLURAL_RESOURCE_NAME = "devices";

export const useSetSynchronizer = (
  onSet: (device: Omit<device.Device, "status">) => void,
): void => {
  const client = Synnax.use();
  useEffect(() => client?.devices.onSet(onSet), [client]);
};

export type RetrieveQuery = device.RetrieveSingleParams;

const BASE_QUERY = { includeStatus: true } as const;

export const createRetrieve = <
  Properties extends z.ZodType<record.Unknown> = z.ZodType<record.Unknown>,
  Make extends z.ZodType<string> = z.ZodString,
  Model extends z.ZodType<string> = z.ZodString,
>(
  schemas?: device.DeviceSchemas<Properties, Make, Model>,
) =>
  Flux.createRetrieve<RetrieveQuery, device.Device<Properties, Make, Model>>({
    name: RESOURCE_NAME,
    normalizeQuery: (query) => ({ ...BASE_QUERY, ...query }),
    retrieve: async ({ client, query }) => {
      if (schemas != null) return await client.devices.retrieve({ ...query, schemas });
      const dev = await client.devices.retrieve(query);
      return dev as unknown as device.Device<Properties, Make, Model>;
    },
    onChange: ({ client, query }, handler) =>
      client.devices.onChange(
        query,
        handler as unknown as query.ChangeHandler<device.Device>,
      ),
    getCached: ({ client, query }) =>
      client.devices.getCached(query) as
        query.Cached<device.Device<Properties, Make, Model>> | undefined,
  });

export const { use, useResult, createResultSelector } = createRetrieve();

/** Compared by variant and message: a heartbeat that changes neither is silenced. */
export const useResultStatus = createResultSelector(
  ({ status }) => status,
  (a, b) => a?.variant === b?.variant && a?.message === b?.message,
);

export const useResultRack = createResultSelector(({ rack }) => rack);

export type ListParams = device.RetrieveMultipleParams;

export const useList = Flux.createList<ListParams, device.Key, device.Device>({
  name: PLURAL_RESOURCE_NAME,
  normalizeQuery: (query) => ({ ...BASE_QUERY, ...query }),
  retrieve: async ({ client, query }) => await client.devices.retrieve(query),
  retrieveByKey: async ({ client, key }) =>
    await client.devices.retrieve({ ...BASE_QUERY, key }),
  onChange: ({ client, query }, handler) => client.devices.onChange(query, handler),
  getCached: ({ client, query }) => client.devices.getCached(query),
});

export type UseDeleteParams = device.Key | device.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.devices.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
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
    device.Device<Properties, Make, Model>
  >({
    name: RESOURCE_NAME,
    verbs: verbs.CREATE,
    update: async ({ data, client }) => {
      const dev =
        schemas != null
          ? await client.devices.create(data, schemas)
          : await client.devices.create(data as device.New);
      return dev as device.Device<Properties, Make, Model>;
    },
  });

export const { useUpdate: useCreate } = createCreate();

export type UseRetrieveGroupParams = Record<string, never>;

export const { use: useGroupID } = Flux.createRetrieve<
  UseRetrieveGroupParams,
  ontology.ID | undefined
>({
  name: "device group",
  retrieve: async ({ client }) => {
    const res = await client.ontology.children.retrieve({ ids: ontology.ROOT_ID });
    return res.find((r) => r.name === "Devices")?.id;
  },
});

export interface RenameParams extends Pick<device.Device, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ data, client, onOptimisticComplete }) => {
    const { key, name } = data;
    await client.devices.rename(key, name, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export const formSchema = device.deviceZ();

/** The schema a form built by {@link createForm} validates against. */
export type FormSchema<S> =
  S extends device.DeviceSchemas<infer P, infer Ma, infer Mo>
    ? ReturnType<typeof device.deviceZ<P, Ma, Mo>>
    : typeof formSchema;

export type FormQuery = RetrieveQuery;

const BLANK_VALUES: z.infer<typeof formSchema> = {
  key: "",
  rack: 0,
  name: "",
  make: "",
  model: "",
  location: "",
  configured: true,
  properties: {},
};

export interface CreateForm {
  (): Flux.UseForm<FormQuery, typeof formSchema>;
  <
    Properties extends z.ZodType<record.Unknown>,
    Make extends z.ZodType<string>,
    Model extends z.ZodType<string>,
  >(
    schemas: device.DeviceSchemas<Properties, Make, Model>,
    initialValues: z.infer<FormSchema<device.DeviceSchemas<Properties, Make, Model>>>,
  ): Flux.UseForm<FormQuery, FormSchema<device.DeviceSchemas<Properties, Make, Model>>>;
}

/**
 * Builds a device form. With vendor schemas the caller supplies initial values that
 * satisfy them; without, the form starts from a blank generic device.
 */
export const createForm: CreateForm = (
  schemas?: device.DeviceSchemas,
  initialValues: z.infer<typeof formSchema> = BLANK_VALUES,
) => {
  const schema = device.deviceZ(schemas);
  // Cached records and streamed set events are parsed generically, so they can carry
  // shapes that predate the vendor's migrations and defaults.
  const parseRecord = (record: unknown): z.infer<typeof schema> | undefined => {
    const parsed = schema.safeParse(record);
    return parsed.success ? parsed.data : undefined;
  };
  return Flux.createForm<FormQuery, typeof schema>({
    name: RESOURCE_NAME,
    schema,
    initialValues,
    normalizeQuery: (query) => ({ ...BASE_QUERY, ...query }),
    retrieve: async ({ query, client }) =>
      schemas != null
        ? await client.devices.retrieve({ ...query, schemas })
        : await client.devices.retrieve(query),
    getCached: ({ client, query }) => {
      const cached = client.devices.getCached(query);
      return clientQuery.isLive(cached) ? parseRecord(cached) : undefined;
    },
    update: async ({ value, client, set }) => {
      const data = value();
      if (primitive.isZero(data.key)) {
        data.key = uuid.create();
        set("key", data.key);
      }
      await client.devices.create(data as device.New, schemas);
    },
    mountListeners: ({ client, query: { key }, reset, set }) => {
      if (primitive.isZero(key)) return [];
      return [
        client.devices.onSet((changed) => {
          if (changed.key !== key) return;
          const parsed = parseRecord(changed);
          if (parsed != null) reset(parsed);
        }),
        client.statuses.onSet((changed) => {
          if (changed.key !== device.statusKey(key)) return;
          set("status", zod.parse(device.statusZ, changed, { label: "device status" }));
        }),
      ];
    },
  });
};

export const useForm = createForm();
