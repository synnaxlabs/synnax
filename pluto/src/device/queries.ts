// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  actions,
  device,
  query as clientQuery,
  type query,
  resource,
  type Synnax as SynnaxClient,
} from "@synnaxlabs/client";
import {
  deep,
  destructor,
  primitive,
  type record,
  unique,
  uuid,
  verbs,
  zod,
} from "@synnaxlabs/x";
import { useEffect } from "react";
import { type z } from "zod";

import { Flux } from "@/flux";
import { flux } from "@/flux/aether";
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

interface CachedParser<
  Properties extends z.ZodType<record.Unknown>,
  Make extends z.ZodType<string>,
  Model extends z.ZodType<string>,
> {
  one: (
    cached: query.Cached<device.Device> | undefined,
  ) => query.Cached<device.Device<Properties, Make, Model>> | undefined;
  many: (
    cached: query.Cached<device.Device[]> | undefined,
  ) => query.Cached<device.Device<Properties, Make, Model>[]> | undefined;
}

/**
 * Parses cached and streamed records through the vendor's schemas, so a typed hook
 * never hands on the untyped records other consumers write into the client cache. A
 * record that fails to parse reads as not cached. Results are memoized per record, as
 * the cache interns its records and Flux compares snapshots by identity.
 */
const createCachedParser = <
  Properties extends z.ZodType<record.Unknown>,
  Make extends z.ZodType<string>,
  Model extends z.ZodType<string>,
>(
  schemas: device.DeviceSchemas<Properties, Make, Model>,
): CachedParser<Properties, Make, Model> => {
  type Typed = device.Device<Properties, Make, Model>;
  const schema = device.deviceZ(schemas);
  const records = new WeakMap<device.Device, Typed | undefined>();
  const deletedOne = new WeakMap<query.Deleted<device.Device>, query.Deleted<Typed>>();
  const deletedMany = new WeakMap<
    query.Deleted<device.Device[]>,
    query.Deleted<Typed[]>
  >();
  const parse = (record: device.Device): Typed | undefined => {
    if (records.has(record)) return records.get(record);
    const result = schema.safeParse(record);
    // deviceZ types each optional schema as a union with its default, so the parsed
    // record narrows to the vendor's type here.
    const typed = result.success ? (result.data as Typed) : undefined;
    records.set(record, typed);
    return typed;
  };
  const parseMany = (devices: device.Device[]): Typed[] | undefined => {
    const typed: Typed[] = [];
    for (const record of devices) {
      const parsed = parse(record);
      if (parsed === undefined) return undefined;
      typed.push(parsed);
    }
    return typed;
  };
  const parseDeleted = <D, T>(
    memo: WeakMap<query.Deleted<D>, query.Deleted<T>>,
    cached: query.Deleted<D>,
    parseCorpse: (corpse: D) => T | undefined,
  ): query.Deleted<T> | undefined => {
    const held = memo.get(cached);
    if (held != null) return held;
    const corpse = parseCorpse(cached.corpse);
    if (corpse === undefined) return undefined;
    const next = new clientQuery.Deleted(corpse, cached.deletedAt);
    memo.set(cached, next);
    return next;
  };
  return {
    one: (cached) => {
      if (cached === undefined) return undefined;
      if (clientQuery.Deleted.matches(cached))
        return parseDeleted(deletedOne, cached, parse);
      return parse(cached);
    },
    many: (cached) => {
      if (cached === undefined) return undefined;
      if (clientQuery.Deleted.matches(cached))
        return parseDeleted(deletedMany, cached, parseMany);
      return parseMany(cached);
    },
  };
};

export const createRetrieve = <
  Properties extends z.ZodType<record.Unknown> = z.ZodType<record.Unknown>,
  Make extends z.ZodType<string> = z.ZodString,
  Model extends z.ZodType<string> = z.ZodString,
>(
  schemas?: device.DeviceSchemas<Properties, Make, Model>,
) => {
  const parser = schemas == null ? null : createCachedParser(schemas);
  return Flux.createRetrieve<RetrieveQuery, device.Device<Properties, Make, Model>>({
    name: RESOURCE_NAME,
    normalizeQuery: (query) => ({ ...BASE_QUERY, ...query }),
    retrieve: async ({ client, query }) => {
      if (schemas != null) return await client.devices.retrieve({ ...query, schemas });
      const dev = await client.devices.retrieve(query);
      return dev as unknown as device.Device<Properties, Make, Model>;
    },
    onChange: ({ client, query }, handler) =>
      parser == null
        ? client.devices
            .watch(query)
            .onChange(handler as unknown as query.ChangeHandler<device.Device>)
        : client.devices.watch(query).onChange((cached) => handler(parser.one(cached))),
    getCached: ({ client, query }) =>
      parser == null
        ? (client.devices.watch(query).getCached() as
            query.Cached<device.Device<Properties, Make, Model>> | undefined)
        : parser.one(client.devices.watch(query).getCached()),
  });
};

export const { use, useResult, createResultSelector } = createRetrieve();

/** Names the devices to retrieve by key or by filter. */
export type RetrieveMultipleQuery = device.RetrieveMultipleParams;

/** Creates retrieve hooks for many devices, each parsed through the given schemas. */
export const createRetrieveMultiple = <
  Properties extends z.ZodType<record.Unknown>,
  Make extends z.ZodType<string>,
  Model extends z.ZodType<string>,
>(
  schemas: device.DeviceSchemas<Properties, Make, Model>,
) => {
  const parser = createCachedParser(schemas);
  const reads = flux.listReads<RetrieveMultipleQuery, device.Device>(
    ({ client, query }) => client.devices.watch(query),
  );
  return Flux.createRetrieve<
    RetrieveMultipleQuery,
    device.Device<Properties, Make, Model>[]
  >({
    name: PLURAL_RESOURCE_NAME,
    // deviceZ types each optional schema as a union with its default, so the parsed
    // records narrow to the vendor's type here.
    retrieve: async (params) =>
      device
        .deviceZ(schemas)
        .array()
        .parse(await reads.retrieve(params)) as device.Device<
        Properties,
        Make,
        Model
      >[],
    onChange: (params, handler) =>
      reads.onChange(params, (cached) => handler(parser.many(cached))),
    getCached: (params) => parser.many(reads.getCached(params)),
  });
};

export const { use: useMultiple, useResult: useResultMultiple } = Flux.createRetrieve<
  RetrieveMultipleQuery,
  device.Device[]
>({
  name: PLURAL_RESOURCE_NAME,
  normalizeQuery: (query) => ({ ...BASE_QUERY, ...query }),
  ...flux.listReads(({ client, query }) => client.devices.watch(query)),
});

export type RetrieveAncestorsQuery = { keys: device.Key[] };

/** Adds the devices to found, and returns the keys above them that it lacks. */
const collect = (
  found: Map<device.Key, device.Device>,
  devices: device.Device[],
): device.Key[] => {
  devices.forEach((d) => found.set(d.key, d));
  return unique.unique(
    devices.flatMap(({ parent }) =>
      parent == null || found.has(parent) ? [] : [parent],
    ),
  );
};

const getCachedAncestry = (
  client: SynnaxClient,
  keys: device.Key[],
): device.Device[] | undefined => {
  const found = new Map<device.Key, device.Device>();
  let next = keys;
  while (next.length > 0) {
    const devices = client.devices.watch(next).getCached();
    if (!clientQuery.isLive(devices)) return undefined;
    next = collect(found, devices);
  }
  return [...found.values()];
};

/**
 * Retrieves the devices with the keys and every device that holds one of them. A
 * change to one of them updates the answer.
 */
export const { useResult: useResultAncestors } = Flux.createRetrieve<
  RetrieveAncestorsQuery,
  device.Device[]
>({
  name: "device ancestors",
  retrieve: async ({ client, query: { keys } }) => {
    const found = new Map<device.Key, device.Device>();
    let next = keys;
    while (next.length > 0) next = collect(found, await client.devices.retrieve(next));
    return [...found.values()];
  },
  onChange: ({ client, query: { keys } }, handler) => {
    const notify = () => handler(getCachedAncestry(client, keys));
    const offSet = client.devices.onSet(notify);
    const offDelete = client.devices.onDelete(notify);
    return () => destructor.unwind(offSet, offDelete);
  },
  getCached: ({ client, query: { keys } }) => getCachedAncestry(client, keys),
});

/** Compared by variant and message: a heartbeat that changes neither is silenced. */
export const useResultStatus = createResultSelector(
  ({ status }) => status,
  (a, b) => a?.variant === b?.variant && a?.message === b?.message,
);

export const useResultMake = createResultSelector(({ make }) => make);

export const useResultLocation = createResultSelector(({ location }) => location);

/** Lists devices with their statuses, also for a device outside the list. */
export const useList = ({
  query,
  ...rest
}: Omit<
  Flux.UseListParams<RetrieveMultipleQuery, device.Device>,
  "itemQuery" | "name"
> = {}): Flux.UseListReturn<device.Key, device.Device> =>
  Flux.useList((client) => client.devices, {
    ...rest,
    name: PLURAL_RESOURCE_NAME,
    query: query === null ? null : { ...BASE_QUERY, ...query },
    itemQuery: (key) => ({ ...BASE_QUERY, key }),
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

export interface RenameParams extends Pick<device.Device, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ data, client }) => {
    const { key, name } = data;
    await client.devices.rename(key, name);
    return data;
  },
});

export const formSchema = device.deviceZ();

export const formPath = deep.path<z.infer<typeof formSchema>>();

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
      const cached = client.devices.watch(query).getCached();
      return clientQuery.isLive(cached) ? parseRecord(cached) : undefined;
    },
    update: async ({ value, client, set, stored }) => {
      const data = value();
      const { status } = data;
      // A status read from the Core names the device as its owner. A beforeSave hook
      // that reports a new status leaves the owner out.
      const reported = status != null && status.owner == null;
      // The key is set before the create, so a failed create leaves it set.
      if (stored == null) {
        if (primitive.isZero(data.key)) {
          data.key = uuid.create();
          set(formPath.key, data.key);
        }
        const sent = reported ? data : device.toUpdate(data);
        await client.devices.create(sent as device.New, schemas);
        return;
      }
      const edits: device.Action[] = actions.diffFields(schema, stored, data, [
        "name",
        "model",
        "location",
        "configured",
        "properties",
      ]);
      if (stored.rack !== data.rack) edits.push(device.setRack({ rack: data.rack }));
      if (edits.length > 0) await client.devices.dispatch(data.key, edits);
      if (reported)
        await client.statuses.set({ ...status, owner: device.resourceID(data.key) });
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
          const { owner } = changed;
          if (owner == null || !resource.idsEqual(owner, device.resourceID(key)))
            return;
          set(
            formPath.status,
            zod.parse(device.statusZ, changed, { label: "device status" }),
          );
        }),
      ];
    },
  });
};

export const useForm = createForm();
