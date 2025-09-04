// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack, type Synnax, task } from "@synnaxlabs/client";
import { type Optional } from "@synnaxlabs/x";
import { useEffect } from "react";
import { z } from "zod";

import { Flux } from "@/flux";

export const FLUX_STORE_KEY = "tasks";

export interface FluxStore
  extends Flux.UnaryStore<task.Key, task.Task, ChangeVariant> {}

interface SubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
}

// Temporary hack that filters the set of commands that should change the
// status of a task to loading.
// Issue: https://linear.app/synnax/issue/SY-2723/fix-handling-of-non-startstop-commands-loading-indicators-in-tasks
const LOADING_COMMANDS = ["start", "stop"];

type ChangeVariant = "config" | "status";

const SET_LISTENER: Flux.ChannelListener<SubStore, typeof task.keyZ> = {
  channel: task.SET_CHANNEL_NAME,
  schema: task.keyZ,
  onChange: async ({ store, changed: key, client }) =>
    store.tasks.set(
      key,
      await client.hardware.tasks.retrieve({ key, includeStatus: true }),
      "config",
    ),
};

const DELETE_LISTENER: Flux.ChannelListener<SubStore, typeof task.keyZ> = {
  channel: task.DELETE_CHANNEL_NAME,
  schema: task.keyZ,
  onChange: ({ store, changed }) => store.tasks.delete(changed),
};

const unknownStatusZ = task.statusZ(z.unknown());

const SET_STATUS_LISTENER: Flux.ChannelListener<SubStore, typeof unknownStatusZ> = {
  channel: task.STATUS_CHANNEL_NAME,
  schema: unknownStatusZ,
  onChange: async ({ store, changed, client }) => {
    const hasTask = store.tasks.has(changed.details.task);
    if (!hasTask) {
      const task = await client.hardware.tasks.retrieve({ key: changed.details.task });
      store.tasks.set(changed.details.task, task, "config");
    }
    store.tasks.set(
      changed.details.task,
      (prev) =>
        prev == null ? prev : client.hardware.tasks.sugar({ ...prev, status: changed }),
      "status",
    );
  },
};

const SET_COMMAND_LISTENER: Flux.ChannelListener<SubStore, typeof task.commandZ> = {
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
  const store = Flux.useStore<SubStore>();
  useEffect(
    () =>
      store.tasks.onSet((task) => {
        if (task.status != null) onStatus(task.status);
      }),
    [store],
  );
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<SubStore> = {
  listeners: [SET_LISTENER, DELETE_LISTENER, SET_STATUS_LISTENER, SET_COMMAND_LISTENER],
};

export interface RetrieveQueryParams {
  key?: task.Key;
  includeStatus?: boolean;
}

const retrieveByKey = async <
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
>(
  client: Synnax,
  store: SubStore,
  params: RetrieveQueryParams & { key: task.Key },
  schemas?: task.Schemas<Type, Config, StatusData>,
): Promise<task.Task<Type, Config, StatusData>> => {
  const cached = store.tasks.get(params.key);
  if (cached != null) return cached as unknown as task.Task<Type, Config, StatusData>;
  const task = await client.hardware.tasks.retrieve<Type, Config, StatusData>({
    ...params,
    includeStatus: true,
    schemas,
  });
  store.tasks.set(params.key, task as unknown as task.Task, "config");
  return task;
};

export const createRetrieveQuery = <
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
>(
  schemas: task.Schemas<Type, Config, StatusData>,
) =>
  Flux.createRetrieve<
    RetrieveQueryParams,
    task.Task<Type, Config, StatusData> | null,
    SubStore
  >({
    name: "Task",
    retrieve: async ({ client, params, store }) => {
      if (params.key == null) return null;
      return await retrieveByKey<Type, Config, StatusData>(
        client,
        store,
        { key: params.key, includeStatus: true },
        schemas,
      );
    },
    mountListeners: ({ store, params: { key }, onChange }) => [
      store.tasks.onSet((task) => {
        if (key == null || task.key !== key) return;
        onChange(task as unknown as task.Task<Type, Config, StatusData>);
      }, key),
    ],
  });

export interface ListParams {
  term?: string;
  offset?: number;
  limit?: number;
}

export const useList = Flux.createList<ListParams, task.Key, task.Task, SubStore>({
  name: "Task",
  retrieveCached: ({ store }) => store.tasks.list(),
  retrieve: async ({ client, params, store }) => {
    const tasks = await client.hardware.tasks.retrieve({
      includeStatus: true,
      ...params,
    });
    tasks.forEach((task) => store.tasks.set(task.key, task, "config"));
    return tasks;
  },

  retrieveByKey: async ({ client, key, store }) =>
    await retrieveByKey(client, store, { key }),
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
  return Flux.createForm<UseFormParams, FormSchema<Type, Config, StatusData>, SubStore>(
    {
      name: "Task",
      schema,
      initialValues: actualInitialValues,
      retrieve: async ({ client, store, params: { key }, reset }): Promise<void> => {
        if (key == null) return;
        const task = await retrieveByKey<Type, Config, StatusData>(
          client,
          store,
          { key },
          schemas,
        );
        reset(taskToFormValues(task.payload));
      },
      update: async ({ client, params, store, ...form }) => {
        const value = form.value();
        const rack = await client.hardware.racks.retrieve({ key: value.rackKey });
        const task = await rack.createTask({
          key: params.key,
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
          "config",
        );
        const updatedValues = taskToFormValues<Type, Config, StatusData>(
          task.payload as task.Payload<Type, Config, StatusData>,
        );
        form.set("key", updatedValues.key);
        form.set("name", updatedValues.name);
        form.set("rackKey", updatedValues.rackKey);
        form.set("type", updatedValues.type);
        form.set("config", updatedValues.config);
        form.set("snapshot", updatedValues.snapshot);
      },
      mountListeners: ({ store, get, reset, set }) => [
        store.tasks.onSet((task, variant) => {
          const prevKey = get<string>("key", { optional: true })?.value;
          if (prevKey == null || prevKey !== task.key) return;
          if (variant === "config") {
            const payload = task.payload as task.Payload<Type, Config, StatusData>;
            reset(taskToFormValues(payload));
          } else if (variant === "status") set("status", task.status);
        }),
      ],
    },
  );
};
