// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { cache, type ontology, type rack, task } from "@synnaxlabs/client";
import { array, type optional } from "@synnaxlabs/x";
import { useCallback } from "react";
import { z } from "zod";

import { Flux } from "@/flux";
import { type Form } from "@/form";
import { useSyncedRef } from "@/hooks/ref";

export const RESOURCE_NAME = "task";
export const PLURAL_RESOURCE_NAME = "tasks";

export type RetrieveQuery = task.RetrieveSingleParams;

const BASE_QUERY = { includeStatus: true };

// Cached answers are untyped; schemas only validate the fetch, so schema-typed
// reads cast the shared cache entries.
export const createRetrieve = <S extends task.Schemas = task.Schemas>(schemas?: S) =>
  Flux.createRetrieve<RetrieveQuery, task.Task<S>>({
    name: RESOURCE_NAME,
    retrieve: async ({ client, query }) =>
      await client.tasks.retrieve({ ...BASE_QUERY, ...query, schemas }),
    subscribe: ({ client, query }, handler) =>
      client.tasks.onChange(query, handler as cache.ChangeHandler<task.Task>),
    getCached: ({ client, query }) =>
      client.tasks.getCached(query) as cache.Cached<task.Task<S>> | undefined,
  });

export const { useRetrieve, useRetrieveObservable, useEnsureRetrieved } =
  createRetrieve();

export interface SelectKeyParams {
  key: task.Key;
}

export const [useSelectName, useGetName] = Flux.createSelector<SelectKeyParams, string>(
  {
    subscribe: ({ client, args: { key } }, notify) =>
      client == null ? () => {} : client.tasks.onChange({ key }, notify),
    select: ({ client, args: { key } }) => {
      const cached = client?.tasks.getCached({ key });
      if (cached == null || cached.variant === "deleted") return "Task";
      return cached.data.name;
    },
  },
);

export const useRetrieveObservableName = ({
  onChange,
  ...params
}: Omit<Flux.UseRetrieveObservableParams<RetrieveQuery, task.Task>, "onChange"> & {
  onChange: (name: string) => void;
}): Flux.UseRetrieveObservableReturn<RetrieveQuery> => {
  const onChangeRef = useSyncedRef(onChange);
  return useRetrieveObservable({
    ...params,
    onChange: useCallback((result) => {
      if (result.variant !== "success" || result.data == null) return;
      onChangeRef.current(result.data.name);
    }, []),
  });
};

export type ListQuery = task.RetrieveMultipleParams;

const listRequest = (query: ListQuery): ListQuery => ({
  ...BASE_QUERY,
  internal: false,
  ...query,
});

export const useList = Flux.createList<ListQuery, task.Key, task.Task>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) =>
    await client.tasks.retrieve(listRequest(query)),
  retrieveByKey: async ({ client, key }) =>
    await client.tasks.retrieve({ ...BASE_QUERY, key }),
  subscribe: ({ client, query }, handler) =>
    client.tasks.onChange(listRequest(query), handler),
  subscribeByKey: ({ client, key }, handler) => client.tasks.onChange({ key }, handler),
  getCached: ({ client, query }) => client.tasks.getCached(listRequest(query)),
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
> extends optional.Optional<task.Payload<S>, "key" | "internal" | "snapshot"> {
  key?: task.Key;
  /** Rack to pre-select when creating a new task. Ignored when key is set, as the
   * rack is already encoded in the task key. */
  rackKey?: rack.Key;
}

export type FormQuery = {
  key?: task.Key;
};

const taskToFormValues = <S extends task.Schemas = task.Schemas>(
  t: InitialValues<S>,
): z.infer<FormSchema<S>> => ({
  key: t.key,
  name: t.name,
  rackKey: t.key == null ? (t.rackKey ?? 0) : task.rackKey(t.key),
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
  return Flux.createForm<FormQuery, FormSchema<S>>({
    name: RESOURCE_NAME,
    schema,
    initialValues: actualInitialValues,
    retrieve: async ({ client, query: { key }, reset }): Promise<void> => {
      if (key == null) return;
      const tsk = await client.tasks.retrieve({ ...BASE_QUERY, key, schemas });
      reset(taskToFormValues(tsk.payload));
    },
    update: async ({ client, ...form }) => {
      const value = form.value();
      const rack = await client.racks.retrieve({ key: value.rackKey });
      const task = await rack.createTask(
        {
          key: value.key,
          name: value.name,
          type: value.type,
          config: value.config,
          status: value.status,
        },
        schemas,
      );
      resetFormValues(form.set, task.payload);
      form.setCurrentStateAsInitialValues();
    },
    mountListeners: ({ client, query: { key }, set }) => {
      if (key == null) return [];
      return client.tasks.onChange({ key }, (result) => {
        if (result?.variant !== "changed") return;
        resetFormValues(set, result.data.payload as task.Payload<S>);
        if (result.data.status != null)
          set(
            "status",
            task.statusZ(z.unknown().optional()).parse(result.data.status),
            RESET_OPTIONS,
          );
      });
    },
  });
};

export type DeleteParams = task.Key | task.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: cache.DELETE_VERBS,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.tasks.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export interface SnapshotPair extends Pick<task.Payload, "key" | "name"> {}

export interface SnapshotParams {
  tasks: SnapshotPair | SnapshotPair[];
  parentID: ontology.ID;
}

export const { useUpdate: useCreateSnapshot } = Flux.createUpdate<SnapshotParams>({
  name: RESOURCE_NAME,
  verbs: cache.SNAPSHOT_VERBS,
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

export interface UseRenameParams extends Pick<task.Payload, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<UseRenameParams>({
  name: RESOURCE_NAME,
  verbs: cache.RENAME_VERBS,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, name } = data;
    await client.tasks.rename(key, name, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
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

const COMMAND_VERBS: cache.Verbs = {
  present: "command",
  participle: "commanding",
  past: "commanded",
};

export const { useUpdate: useCommand } = Flux.createUpdate<CommandParams>({
  name: PLURAL_RESOURCE_NAME,
  verbs: COMMAND_VERBS,
  update: async ({ data, client }) => {
    const commands = array.toArray(data);
    const keys = commands.map(({ task }) => task);
    // Warm the cache so command listeners can update the tasks' statuses.
    await client.tasks.retrieve({ keys, includeStatus: true });
    await client.tasks.executeCommand({ commands });
    return data;
  },
});
