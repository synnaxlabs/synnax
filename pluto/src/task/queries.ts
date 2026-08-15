// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, query, rack, task } from "@synnaxlabs/client";
import { array, type optional, verbs } from "@synnaxlabs/x";
import { z } from "zod";

import { Flux } from "@/flux";
import { type Form } from "@/form";

export const RESOURCE_NAME = "task";
export const PLURAL_RESOURCE_NAME = "tasks";

export type RetrieveQuery = task.RetrieveSingleParams;

const BASE_QUERY = { includeStatus: true };

// Cached answers are untyped; schemas only validate the fetch, so schema-typed
// reads cast the shared cache entries.
export const createRetrieve = <S extends task.Schemas = task.Schemas>(schemas?: S) =>
  Flux.createRetrieve<RetrieveQuery, task.Task<S>>({
    name: RESOURCE_NAME,
    normalizeQuery: (query) => ({ ...BASE_QUERY, ...query }),
    retrieve: async ({ client, query }) =>
      await client.tasks.retrieve({ ...query, schemas }),
    onChange: ({ client, query }, handler) =>
      client.tasks.onChange(query, handler as query.ChangeHandler<task.Task>),
    getCached: ({ client, query }) =>
      client.tasks.getCached(query) as query.Cached<task.Task<S>> | undefined,
  });

export const { use, useEnsure, useResult, useTombstone, createSelector } =
  createRetrieve();

export interface KeyParams {
  key: task.Key;
}

export const useName = createSelector(({ name }) => name);

export type ListQuery = task.RetrieveMultipleParams;

export const useList = Flux.createList<ListQuery, task.Key, task.Task>({
  name: PLURAL_RESOURCE_NAME,
  normalizeQuery: (query) => ({ ...BASE_QUERY, internal: false, ...query }),
  retrieve: async ({ client, query }) => await client.tasks.retrieve(query),
  retrieveByKey: async ({ client, key }) =>
    await client.tasks.retrieve({ ...BASE_QUERY, key }),
  onChange: ({ client, query }, handler) => client.tasks.onChange(query, handler),
  onChangeByKey: ({ client, key }, handler) => client.tasks.onChange(key, handler),
  getCached: ({ client, query }) => client.tasks.getCached(query),
});

const createFormSchema = <S extends task.Schemas = task.Schemas>(
  schemas: S,
): FormSchema<S> =>
  z.object({
    key: task.keyZ.optional(),
    name: z.string(),
    rack: rack.keyZ,
    type: schemas.type,
    snapshot: z.boolean(),
    config: schemas.config,
    configHash: z.string(),
    status: task.statusZ(schemas.statusData).optional().nullable(),
  }) as unknown as FormSchema<S>;

export interface FormSchema<S extends task.Schemas = task.Schemas> extends z.ZodType<{
  key?: task.Key;
  name: string;
  rack: rack.Key;
  type: z.infer<S["type"]>;
  snapshot: boolean;
  config: z.infer<S["config"]>;
  configHash: string;
  status?: task.Status<S["statusData"]>;
}> {}

export interface CreateFormParams<S extends task.Schemas = task.Schemas> {
  schemas: S;
  initialValues: InitialValues<S>;
}

export interface InitialValues<
  S extends task.Schemas = task.Schemas,
> extends optional.Optional<
  task.Payload<S>,
  "key" | "rack" | "internal" | "snapshot" | "configHash"
> {}

export type FormQuery = {
  key: task.Key;
};

export const toFormValues = <S extends task.Schemas = task.Schemas>(
  t: InitialValues<S>,
): z.infer<FormSchema<S>> => ({
  key: t.key,
  name: t.name,
  rack: t.rack ?? 0,
  type: t.type,
  config: t.config,
  configHash: t.configHash ?? "",
  status: t.status,
  snapshot: t.snapshot ?? false,
});

const RESET_OPTIONS: Form.SetOptions = { markTouched: false };

export const createForm = <S extends task.Schemas = task.Schemas>({
  schemas,
  initialValues,
}: CreateFormParams<S>) => {
  const schema = createFormSchema(schemas);
  const actualInitialValues = toFormValues(initialValues);
  return Flux.createForm<FormQuery, FormSchema<S>>({
    name: RESOURCE_NAME,
    schema,
    initialValues: actualInitialValues,
    normalizeQuery: (query) => ({ ...BASE_QUERY, ...query }),
    retrieve: async ({ client, query: q }) =>
      toFormValues((await client.tasks.retrieve({ ...q, schemas })).payload),
    getCached: ({ client, query: q }) => {
      const cached = client.tasks.getCached(q);
      if (!query.isLive(cached) || cached.status == null) return undefined;
      return toFormValues(cached.payload as task.Payload<S>);
    },
    update: async ({ client, ...form }) => {
      const value = form.value();
      const created = await client.tasks.create(
        {
          key: value.key,
          rack: value.rack,
          name: value.name,
          type: value.type,
          config: value.config,
          snapshot: value.snapshot,
          status: value.status ?? undefined,
        },
        schemas,
      );
      // Only server-assigned fields are reset from the response: resetting an
      // edited field would clobber edits typed while this save was in flight.
      form.set("key", created.key, RESET_OPTIONS);
      form.set("configHash", created.configHash, RESET_OPTIONS);
      form.setCurrentStateAsInitialValues();
    },
    mountListeners: ({ client, query: { key }, set }) =>
      client.tasks.onChange(key, (result) => {
        if (!query.isLive(result)) return;
        // Metadata only: config changes come solely from this form's own
        // saves, and resetting it would clobber in-flight autosave edits.
        const payload = result.payload as task.Payload<S>;
        set("name", payload.name, RESET_OPTIONS);
        set("rack", payload.rack, RESET_OPTIONS);
        set("snapshot", payload.snapshot, RESET_OPTIONS);
        set("configHash", payload.configHash, RESET_OPTIONS);
        if (result.status != null)
          set(
            "status",
            task.statusZ(z.unknown().optional()).parse(result.status),
            RESET_OPTIONS,
          );
      }),
  });
};

export type DeleteParams = task.Key | task.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
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
  verbs: verbs.SNAPSHOT,
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
  verbs: verbs.RENAME,
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

const COMMAND_VERBS: verbs.Verbs = {
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
