// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, type rack, task } from "@synnaxlabs/client";
import { array, type optional, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";

import { Flux } from "@/flux";
import { type Form } from "@/form";
import { type Label } from "@/label";
import { Ontology } from "@/ontology";
import { state } from "@/state";
import { Status } from "@/status";

export const FLUX_STORE_KEY = "tasks";
export const RESOURCE_NAME = "task";
export const PLURAL_RESOURCE_NAME = "tasks";

export interface FluxStore extends Flux.UnaryStore<
  task.Key,
  Omit<task.Task, "status">
> {}

export interface FluxSubStore extends Ontology.FluxSubStore, Label.FluxSubStore {
  [FLUX_STORE_KEY]: FluxStore;
  [Status.FLUX_STORE_KEY]: Status.FluxStore;
}

// Temporary hack that filters the set of commands that should change the
// status of a task to loading.
// Issue: https://linear.app/synnax/issue/SY-2723/fix-handling-of-non-startstop-commands-loading-indicators-in-tasks
const LOADING_COMMANDS = ["start", "stop"];

const SET_LISTENER: Flux.ChannelListener<FluxSubStore, typeof task.keyZ> = {
  channel: task.SET_CHANNEL_NAME,
  schema: task.keyZ,
  onChange: async ({ store, changed: key, client }) =>
    store.tasks.set(key, await client.tasks.retrieve({ key, includeStatus: true })),
};

const DELETE_LISTENER: Flux.ChannelListener<FluxSubStore, typeof task.keyZ> = {
  channel: task.DELETE_CHANNEL_NAME,
  schema: task.keyZ,
  onChange: ({ store, changed }) => store.tasks.delete(changed),
};

const SET_COMMAND_LISTENER: Flux.ChannelListener<FluxSubStore, typeof task.commandZ> = {
  channel: task.COMMAND_CHANNEL_NAME,
  schema: task.commandZ,
  onChange: ({ store, changed }) => {
    store.statuses.set(task.statusKey(changed.task), (prev) => {
      if (prev == null || !LOADING_COMMANDS.includes(changed.type)) return prev;
      const status: task.Status = {
        key: task.statusKey(changed.task),
        name: "Task Status",
        time: TimeStamp.now(),
        variant: "loading",
        message: `Running ${changed.type} command...`,
        details: { task: changed.task, running: true, data: {} },
      };
      return status;
    });
  },
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<FluxSubStore> = {
  listeners: [SET_LISTENER, DELETE_LISTENER, SET_COMMAND_LISTENER],
};

export type RetrieveQuery = task.RetrieveSingleParams;

const BASE_QUERY: Partial<RetrieveQuery> = { includeStatus: true };

export const retrieveSingle = async <S extends task.Schemas = task.Schemas>({
  query,
  schemas,
  client,
  store,
}: Flux.RetrieveParams<RetrieveQuery, FluxSubStore> & {
  schemas?: S;
}): Promise<task.Task<S>> => {
  if ("key" in query && query.key != null) {
    const cached = store.tasks.get(query.key.toString());
    if (cached != null) {
      const tsk = cached as unknown as task.Task<S>;
      const detailsSchema = task.statusDetailsZ(schemas?.statusData ?? z.unknown());
      tsk.status = await Status.retrieveSingle({
        store,
        client,
        query: { key: task.statusKey(query.key.toString()) },
        detailsSchema,
      });
      return tsk;
    }
  }
  const tsk = await client.tasks.retrieve({ ...BASE_QUERY, ...query, schemas });
  store.tasks.set(tsk.key.toString(), tsk as unknown as task.Task);
  if (tsk.status != null) store.statuses.set(tsk.status);
  return tsk;
};

export const createRetrieve = <S extends task.Schemas = task.Schemas>(schemas?: S) =>
  Flux.createRetrieve<RetrieveQuery, task.Task<S> | null, FluxSubStore>({
    name: RESOURCE_NAME,
    retrieve: async (args) => await retrieveSingle({ ...args, schemas }),
    mountListeners: ({ store, query, onChange, client }) => {
      if (!("key" in query) || query.key == null) return [];
      return [
        store.tasks.onSet((task) => {
          if ("key" in query && query.key != null && task.key === query.key)
            onChange(task as unknown as task.Task<S>);
        }, query.key.toString()),
        store.statuses.onSet(
          (status) => {
            const parsed = task
              .statusZ(schemas?.statusData ?? z.unknown())
              .parse(status);
            onChange((prev) => {
              if (prev == null) return null;
              return client.tasks.sugar({ ...prev.payload, status: parsed });
            });
          },
          task.statusKey(query.key as task.Key),
        ),
      ];
    },
  });

export const { useRetrieve } = createRetrieve();

export interface ListQuery extends task.RetrieveMultipleParams {}

const unknownStatusZ = task.statusZ(z.unknown());

export const useList = Flux.createList<ListQuery, task.Key, task.Task, FluxSubStore>({
  name: PLURAL_RESOURCE_NAME,
  retrieveCached: ({ store }) => {
    const tasks = store.tasks.list();
    return tasks.map((t) => {
      const status = store.statuses.get(task.statusKey(t.key.toString()));
      const tsk: task.Task = t;
      tsk.status = status as task.Status<z.ZodUnknown>;
      return tsk;
    });
  },
  retrieve: async ({ client, query, store }) => {
    const tasks = await client.tasks.retrieve({
      ...BASE_QUERY,
      internal: false,
      ...query,
    });
    store.tasks.set(tasks);
    const statuses = tasks.map((t) => t.status).filter((s) => s != null);
    store.statuses.set(statuses);
    return tasks;
  },
  retrieveByKey: async ({ key, ...args }) =>
    await retrieveSingle({ ...args, query: { key } }),
  mountListeners: ({ store, onChange, onDelete, client }) => [
    store.tasks.onSet((task) => onChange(task.key, task)),
    store.tasks.onDelete(onDelete),
    store.statuses.onSet((status) => {
      const parsed = unknownStatusZ.safeParse(status);
      if (!parsed.success) return;
      onChange(
        parsed.data.details.task,
        state.skipNull((p) => client.tasks.sugar({ ...p, status: parsed.data })),
      );
    }),
  ],
});

const createFormSchema = <S extends task.Schemas = task.Schemas>(
  schemas: S,
): FormSchema<S> =>
  z.object({
    key: task.keyZ.optional(),
    name: z.string(),
    rackKey: z.number(),
    type: schemas.type,
    snapshot: z.boolean(),
    config: schemas.config,
    status: task.statusZ(schemas.statusData).optional().nullable(),
  }) as unknown as FormSchema<S>;

export interface FormSchema<S extends task.Schemas = task.Schemas> extends z.ZodType<{
  key?: task.Key;
  name: string;
  rackKey: rack.Key;
  type: z.infer<S["type"]>;
  snapshot: boolean;
  config: z.infer<S["config"]>;
  status?: task.Status<S["statusData"]>;
}> {}

export interface CreateFormParams<S extends task.Schemas = task.Schemas> {
  schemas: S;
  initialValues: InitialValues<S>;
}

export interface InitialValues<
  S extends task.Schemas = task.Schemas,
> extends optional.Optional<task.Payload<S>, "key"> {
  key?: task.Key;
}

export interface FormQuery {
  key?: task.Key;
}

const taskToFormValues = <S extends task.Schemas = task.Schemas>(
  t: InitialValues<S>,
): z.infer<FormSchema<S>> => ({
  key: t.key,
  name: t.name,
  rackKey: t.key == null ? 0 : task.rackKey(t.key),
  type: t.type,
  config: t.config,
  status: t.status,
  snapshot: t.snapshot ?? false,
});

const RESET_OPTIONS: Form.SetOptions = { markTouched: false };

const resetFormValues = <S extends task.Schemas = task.Schemas>(
  set: Form.UseReturn<FormSchema<S>>["set"],
  payload: task.Payload<S>,
) => {
  const values = taskToFormValues(payload);
  set("key", values.key, RESET_OPTIONS);
  set("name", values.name, RESET_OPTIONS);
  set("type", values.type, RESET_OPTIONS);
  set("rackKey", values.rackKey, RESET_OPTIONS);
  set("config", values.config, RESET_OPTIONS);
  set("snapshot", values.snapshot, RESET_OPTIONS);
};

export const createForm = <S extends task.Schemas = task.Schemas>({
  schemas,
  initialValues,
}: CreateFormParams<S>) => {
  const schema = createFormSchema(schemas);
  const actualInitialValues = taskToFormValues(initialValues);
  return Flux.createForm<FormQuery, FormSchema<S>, FluxSubStore>({
    name: RESOURCE_NAME,
    schema,
    initialValues: actualInitialValues,
    retrieve: async (args): Promise<void> => {
      const {
        query: { key },
        reset,
      } = args;
      if (key == null) return;
      const task = await retrieveSingle({ ...args, query: { key }, schemas });
      reset(taskToFormValues(task.payload));
    },
    update: async ({ client, store, ...form }) => {
      const value = form.value();
      const rack = await client.racks.retrieve({ key: value.rackKey });
      const task = await rack.createTask(
        {
          key: value.key,
          name: value.name,
          type: value.type,
          config: value.config,
          status: value.status as task.NewStatus<S["statusData"]>,
        },
        schemas,
      );
      store.tasks.set(task as unknown as Omit<task.Task, "status">);
      resetFormValues(form.set, task.payload);
      form.setCurrentStateAsInitialValues();
    },
    mountListeners: ({ store, get, set }) => [
      store.tasks.onSet((task) => {
        const prevKey = get<string>("key", { optional: true })?.value;
        if (prevKey == null || prevKey !== task.key) return;
        resetFormValues(set, task.payload);
      }),
      store.statuses.onSet((status) => {
        const prevKey = get<string>("key", { optional: true })?.value;
        if (prevKey == null || status.key !== task.statusKey(prevKey)) return;
        set("status", task.statusZ(z.unknown()).parse(status), RESET_OPTIONS);
      }),
    ],
  });
};

export type DeleteParams = task.Key | task.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const keys = array.toArray(data);
    const ids = task.ontologyID(keys);
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.push(store.relationships.delete(relFilter));
    rollbacks.push(store.resources.delete(ontology.idToString(ids)));
    rollbacks.push(store.tasks.delete(keys));
    const statusKeys = keys.map((key) => task.statusKey(key));
    rollbacks.push(store.statuses.delete(statusKeys));
    // Task client will automatically handle the deletion of the statuses.
    await client.tasks.delete(keys);
    return data;
  },
});

export interface SnapshotPair extends Pick<task.Payload, "key" | "name"> {}

export interface SnapshotParams {
  tasks: SnapshotPair | SnapshotPair[];
  parentID: ontology.ID;
}

export const { useUpdate: useCreateSnapshot } = Flux.createUpdate<
  SnapshotParams,
  FluxSubStore
>({
  name: RESOURCE_NAME,
  verbs: Flux.SNAPSHOT_VERBS,
  update: async ({ client, data }) => {
    const { tasks: taskPairs, parentID } = data;
    const tasks = await Promise.all(
      array
        .toArray(taskPairs)
        .map(({ key, name }) => client.tasks.copy(key, `${name} (Snapshot)`, true)),
    );
    const otgIDs = tasks.map(({ ontologyID }) => ontologyID);
    await client.ontology.addChildren(parentID, ...otgIDs);
    return data;
  },
});

export interface UseRenameArgs extends Pick<task.Payload, "key" | "name"> {}

export const rename = async (
  params: Flux.UpdateParams<UseRenameArgs, FluxSubStore>,
): Promise<UseRenameArgs> => {
  const {
    client,
    data,
    rollbacks,
    store,
    data: { key, name },
  } = params;
  rollbacks.push(
    store.tasks.set(
      key,
      state.skipUndefined((p) => client.tasks.sugar({ ...p.payload, name })),
    ),
  );
  rollbacks.push(Ontology.renameFluxResource(store, task.ontologyID(key), name));
  const t = await retrieveSingle({ ...params, query: { key } });
  await client.tasks.create({ ...t.payload, name });
  return data;
};

export const { useUpdate: useRename } = Flux.createUpdate<UseRenameArgs, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: rename,
});

export type CommandParams = task.NewCommand | task.NewCommand[];

const START_STOP_COMMANDS = new Set(["stop", "start"]);

export const shouldExecuteCommand = <StatusData extends z.ZodType = z.ZodNever>(
  status: task.Status<StatusData>,
  command: string,
): boolean => {
  if (!START_STOP_COMMANDS.has(command)) return true;
  return (
    (status.details.running && command === "stop") ||
    (!status.details.running && command === "start")
  );
};

const COMMAND_VERBS: Flux.Verbs = {
  present: "command",
  participle: "commanding",
  past: "commanded",
};

export const { useUpdate: useCommand } = Flux.createUpdate<CommandParams, FluxSubStore>(
  {
    name: PLURAL_RESOURCE_NAME,
    verbs: COMMAND_VERBS,
    update: async ({ data, client, store }) => {
      const commands = array.toArray(data);
      const keys = commands.map(({ task }) => task);
      const tasks = store.tasks.get(keys);
      if (tasks.length < keys.length) {
        const existingKeys = tasks.map(({ key }) => key);
        const difference = new Set(keys).difference(new Set(existingKeys));
        const fetchedTasks = await client.tasks.retrieve({
          keys: Array.from(difference),
          includeStatus: true,
        });
        store.tasks.set(fetchedTasks);
      }
      await client.tasks.executeCommand({ commands });
      return data;
    },
  },
);
