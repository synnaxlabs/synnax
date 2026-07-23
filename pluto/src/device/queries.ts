// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, ontology, type query } from "@synnaxlabs/client";
import { primitive, type record, uuid } from "@synnaxlabs/x";
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
    retrieve: async ({ client, query }) => {
      if (schemas != null)
        return await client.devices.retrieve({ ...BASE_QUERY, ...query, schemas });
      const dev = await client.devices.retrieve({ ...BASE_QUERY, ...query });
      return dev as unknown as device.Device<Properties, Make, Model>;
    },
    subscribe: ({ client, query }, handler) =>
      client.devices.onChange(
        { ...BASE_QUERY, ...query },
        handler as unknown as query.ChangeHandler<device.Device>,
      ),
    getCached: ({ client, query }) =>
      client.devices.getCached({ ...BASE_QUERY, ...query }) as
        query.Cached<device.Device<Properties, Make, Model>> | undefined,
  });

export const {
  useRetrieve,
  useRetrieveStateful: useStatefulRetrieve,
  useRetrieveEffect,
} = createRetrieve();

export type ListParams = device.RetrieveMultipleParams;

export const useList = Flux.createList<ListParams, device.Key, device.Device>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) =>
    await client.devices.retrieve({ ...BASE_QUERY, ...query }),
  retrieveByKey: async ({ client, key }) =>
    await client.devices.retrieve({ ...BASE_QUERY, key }),
  subscribe: ({ client, query }, handler) =>
    client.devices.onChange({ ...BASE_QUERY, ...query }, handler),
  getCached: ({ client, query }) =>
    client.devices.getCached({ ...BASE_QUERY, ...query }),
});

export type UseDeleteParams = device.Key | device.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteParams>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
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
    verbs: Flux.CREATE_VERBS,
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

export const { useRetrieve: useRetrieveGroupID } = Flux.createRetrieve<
  UseRetrieveGroupParams,
  ontology.ID | undefined
>({
  name: "Device Group",
  retrieve: async ({ client }) => {
    const res = await client.ontology.retrieveChildren(ontology.ROOT_ID);
    return res.find((r) => r.name === "Devices")?.id;
  },
});

export interface RenameParams extends Pick<device.Device, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ data, client, onOptimisticComplete }) => {
    const { key, name } = data;
    await client.devices.rename(key, name, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
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
  Flux.createForm<FormQuery, typeof formSchema>({
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
    retrieve: async ({ query, client, reset, set }) => {
      if (primitive.isZero(query.key)) {
        set("key", uuid.create());
        return;
      }
      reset(
        schemas != null
          ? await client.devices.retrieve({ ...BASE_QUERY, ...query, schemas })
          : await client.devices.retrieve({ ...BASE_QUERY, ...query }),
      );
    },
    update: async ({ value, client }) => {
      const data = value();
      if (schemas != null)
        await client.devices.create(
          data as device.New<Properties, Make, Model>,
          schemas,
        );
      else await client.devices.create(data);
    },
    mountListeners: ({ client, query: { key }, reset, set }) => {
      if (primitive.isZero(key)) return [];
      const schema = device.deviceZ(schemas);
      return [
        // Streamed set events are parsed generically, so they can carry shapes
        // that predate the vendor's migrations and defaults. Only reset the form
        // when the event satisfies the vendor schemas.
        client.devices.onSet((changed) => {
          if (changed.key !== key) return;
          const parsed = schema.safeParse(changed);
          if (parsed.success) reset(parsed.data as z.infer<typeof formSchema>);
        }),
        client.statuses.onSet((changed) => {
          if (changed.key !== device.statusKey(key)) return;
          set("status", device.statusZ.parse(changed));
        }),
      ];
    },
  });

export const useForm = createForm();
