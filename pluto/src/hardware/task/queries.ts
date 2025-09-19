// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, type rack, type Synnax, task } from "@synnaxlabs/client";
import { array, type Optional } from "@synnaxlabs/x";
import { useEffect } from "react";
import { z } from "zod";

import { Flux } from "@/flux";
import { Ontology } from "@/ontology";

export const FLUX_STORE_KEY = "tasks";

export interface FluxStore
  extends Flux.UnaryStore<task.Key, task.Task, ChangeVariant> {}

export interface FluxSubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.ResourceFluxStore;
}

// Temporary hack that filters the set of commands that should change the
// status of a task to loading.
// Issue: https://linear.app/synnax/issue/SY-2723/fix-handling-of-non-startstop-commands-loading-indicators-in-tasks
const LOADING_COMMANDS = ["start", "stop"];

type ChangeVariant = "payload" | "status";

const SET_LISTENER: Flux.ChannelListener<FluxSubStore, typeof task.keyZ> = {
  channel: task.SET_CHANNEL_NAME,
  schema: task.keyZ,
  onChange: async ({ store, changed: key, client }) =>
    store.tasks.set(
      key,
      await client.hardware.tasks.retrieve({ key, includeStatus: true }),
      "payload",
    ),
};

const DELETE_LISTENER: Flux.ChannelListener<FluxSubStore, typeof task.keyZ> = {
  channel: task.DELETE_CHANNEL_NAME,
  schema: task.keyZ,
  onChange: ({ store, changed }) => store.tasks.delete(changed),
};

const unknownStatusZ = task.statusZ(z.unknown());

const SET_STATUS_LISTENER: Flux.ChannelListener<FluxSubStore, typeof unknownStatusZ> = {
  channel: task.STATUS_CHANNEL_NAME,
  schema: unknownStatusZ,
  onChange: async ({ store, changed, client }) => {
    const hasTask = store.tasks.has(changed.details.task);
    if (!hasTask) {
      const task = await client.hardware.tasks.retrieve({ key: changed.details.task });
      store.tasks.set(changed.details.task, task, "payload");
    }
    store.tasks.set(
      changed.details.task,
      (prev) =>
        prev == null ? prev : client.hardware.tasks.sugar({ ...prev, status: changed }),
      "status",
    );
  },
};

const SET_COMMAND_LISTENER: Flux.ChannelListener<FluxSubStore, typeof task.commandZ> = {
  channel: task.COMMAND_CHANNEL_NAME,
  schema: task.commandZ,
  onChange: ({ store, changed, client }) =>
    store.tasks.set(
      changed.task,
      (prev) => {
        if (prev == null || !LOADING_COMMANDS.includes(changed.type)) return prev;
        return client.hardware.tasks.sugar({
          ...prev,
          status: {
            ...prev.status,
            variant: "loading",
            message: `Running ${changed.type} command...`,
            details: { task: changed.task, running: true, data: {} },
          },
        } as task.Task);
      },
      "status",
    ),
};

export const useStatusSynchronizer = (
  onStatus: (status: task.Status) => void,
): void => {
  const store = Flux.useStore<FluxSubStore>();
  useEffect(
    () =>
      store.tasks.onSet((task) => {
        if (task.status != null) onStatus(task.status);
      }),
    [store],
  );
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<FluxSubStore> = {
  listeners: [SET_LISTENER, DELETE_LISTENER, SET_STATUS_LISTENER, SET_COMMAND_LISTENER],
};

export type UseRetrieveParams = task.SingleRetrieveArgs;

export const retrieveSingle = async <
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
>(
  client: Synnax,
  store: FluxSubStore,
  params: UseRetrieveParams,
  schemas?: task.Schemas<Type, Config, StatusData>,
): Promise<task.Task<Type, Config, StatusData>> => {
  if ("key" in params && params.key != null) {
    const cached = store.tasks.get(params.key.toString());
    if (cached != null) return cached as unknown as task.Task<Type, Config, StatusData>;
  }
  const task = await client.hardware.tasks.retrieve<Type, Config, StatusData>({
    ...params,
    includeStatus: true,
    schemas,
  });
  store.tasks.set(task.key.toString(), task as unknown as task.Task, "payload");
  return task;
};

export const createRetrieve = <
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
>(
  schemas?: task.Schemas<Type, Config, StatusData>,
) =>
  Flux.createRetrieve<
    UseRetrieveParams,
    task.Task<Type, Config, StatusData> | null,
    FluxSubStore
  >({
    name: "Task",
    retrieve: async ({ client, params, store }) =>
      await retrieveSingle<Type, Config, StatusData>(client, store, params, schemas),
    mountListeners: ({ store, params, onChange }) => {
      if (!("key" in params) || params.key == null) return [];
      return [
        store.tasks.onSet((task) => {
          if ("key" in params && params.key != null && task.key === params.key)
            onChange(task as unknown as task.Task<Type, Config, StatusData>);
        }, params.key.toString()),
      ];
    },
  });

export const { useRetrieve } = createRetrieve();

export interface ListParams {
  term?: string;
  offset?: number;
  limit?: number;
}

export const useList = Flux.createList<ListParams, task.Key, task.Task, FluxSubStore>({
  name: "Task",
  retrieveCached: ({ store }) => store.tasks.list(),
  retrieve: async ({ client, params, store }) => {
    const tasks = await client.hardware.tasks.retrieve({
      includeStatus: true,
      ...params,
    });
    tasks.forEach((task) => store.tasks.set(task.key, task, "payload"));
    return tasks;
  },

  retrieveByKey: async ({ client, key, store }) =>
    await retrieveSingle(client, store, { key }),
  mountListeners: ({ store, onChange, onDelete }) => [
    store.tasks.onSet((task) => onChange(task.key, task)),
    store.tasks.onDelete(onDelete),
  ],
});

const createSchema = <
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
>(
  schemas: task.Schemas<Type, Config, StatusData>,
): FormSchema<Type, Config, StatusData> =>
  z.object({
    key: task.keyZ.optional(),
    name: z.string(),
    rackKey: z.number(),
    type: schemas.typeSchema,
    snapshot: z.boolean(),
    config: schemas.configSchema,
    status: task.statusZ(schemas.statusDataSchema).optional().nullable(),
  }) as unknown as FormSchema<Type, Config, StatusData>;

export type FormSchema<
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
> = z.ZodType<{
  key?: task.Key;
  name: string;
  rackKey: rack.Key;
  type: z.infer<Type>;
  snapshot: boolean;
  config: z.infer<Config>;
  status?: z.infer<ReturnType<typeof task.statusZ<StatusData>>>;
}>;

export interface CreateFormArgs<
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
> {
  schemas: task.Schemas<Type, Config, StatusData>;
  initialValues: InitialValues<Type, Config, StatusData>;
}

export interface InitialValues<
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
> extends Optional<task.Payload<Type, Config, StatusData>, "key"> {
  key?: task.Key;
}

export interface UseFormParams {
  key?: task.Key;
}

const taskToFormValues = <
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
>(
  t: InitialValues<Type, Config, StatusData>,
): z.infer<FormSchema<Type, Config, StatusData>> => ({
  key: t.key,
  name: t.name,
  rackKey: t.key == null ? 0 : task.rackKey(t.key),
  type: t.type,
  config: t.config,
  status: t.status,
  snapshot: t.snapshot ?? false,
});

export const createForm = <
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
>({
  schemas,
  initialValues,
}: CreateFormArgs<Type, Config, StatusData>) => {
  const schema = createSchema<Type, Config, StatusData>(schemas);
  const actualInitialValues = taskToFormValues<Type, Config, StatusData>(initialValues);
  return Flux.createForm<
    UseFormParams,
    FormSchema<Type, Config, StatusData>,
    FluxSubStore
  >({
    name: "Task",
    schema,
    initialValues: actualInitialValues,
    retrieve: async ({ client, store, params: { key }, reset }): Promise<void> => {
      if (key == null) return;
      const task = await retrieveSingle<Type, Config, StatusData>(
        client,
        store,
        { key },
        schemas,
      );
      reset(taskToFormValues(task.payload));
    },
    update: async ({ client, store, ...form }) => {
      const value = form.value();
      const rack = await client.hardware.racks.retrieve({ key: value.rackKey });
      const task = await rack.createTask({
        key: value.key,
        name: value.name,
        type: value.type,
        config: value.config,
      });
      store.tasks.set(
        task.key,
        (p) => {
          if (p == null) return p;
          task.status = p.status;
          return task;
        },
        "payload",
      );
      const updatedValues = taskToFormValues<Type, Config, StatusData>(
        task.payload as task.Payload<Type, Config, StatusData>,
      );
      form.set("key", updatedValues.key);
      form.set("name", updatedValues.name);
      form.set("rackKey", updatedValues.rackKey);
      form.set("type", updatedValues.type);
      form.set("payload", updatedValues.config);
      form.set("snapshot", updatedValues.snapshot);
    },
    mountListeners: ({ store, get, reset, set }) => [
      store.tasks.onSet((task, variant) => {
        const prevKey = get<string>("key", { optional: true })?.value;
        if (prevKey == null || prevKey !== task.key) return;
        if (variant === "payload") {
          const payload = task.payload as task.Payload<Type, Config, StatusData>;
          reset(taskToFormValues(payload));
        } else if (variant === "status") set("status", task.status);
      }),
    ],
  });
};

export type UseDeleteArgs = task.Key | task.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteArgs, FluxSubStore>({
  name: "Task",
  update: async ({ client, value, store, rollbacks }) => {
    const keys = array.toArray(value);
    const ids = keys.map((key) => task.ontologyID(key));
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.add(store.relationships.delete(relFilter));
    rollbacks.add(store.resources.delete(ontology.idToString(ids)));
    rollbacks.add(store.tasks.delete(keys));
    await client.hardware.tasks.delete(keys);
    return value;
  },
});

export interface SnapshotPair extends Pick<task.Payload, "key" | "name"> {}

export interface UseSnapshotArgs {
  tasks: SnapshotPair | SnapshotPair[];
  parentID: ontology.ID;
}

export const { useUpdate: useCreateSnapshot } = Flux.createUpdate<UseSnapshotArgs>({
  name: "Task",
  update: async ({ client, value }) => {
    const { tasks: taskPairs, parentID } = value;
    const tasks = await Promise.all(
      array
        .toArray(taskPairs)
        .map(({ key, name }) =>
          client.hardware.tasks.copy(key, `${name} (Snapshot)`, true),
        ),
    );
    const otgIDs = tasks.map(({ ontologyID }) => ontologyID);
    await client.ontology.addChildren(parentID, ...otgIDs);
    return value;
  },
});

export interface UseRenameArgs {
  key: task.Key;
  name: string;
}

export const { useUpdate: useRename } = Flux.createUpdate<UseRenameArgs, FluxSubStore>({
  name: "Task",
  update: async ({ client, value, rollbacks, store }) => {
    const { key, name } = value;
    rollbacks.add(
      store.tasks.set(
        key,
        (p) =>
          p == null ? undefined : client.hardware.tasks.sugar({ ...p.payload, name }),
        "payload",
      ),
    );
    rollbacks.add(Ontology.renameFluxResource(store, task.ontologyID(key), name));
    const t = await retrieveSingle(client, store, { key });
    await client.hardware.tasks.create({ ...t.payload, name });
    return value;
  },
});
