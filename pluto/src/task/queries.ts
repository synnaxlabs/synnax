// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, query, rack, type Synnax, task } from "@synnaxlabs/client";
import { type Form } from "@synnaxlabs/lyra/form";
import { array, type optional, verbs, zod } from "@synnaxlabs/x";
import { z } from "zod";

import { Flux } from "@/flux";

export const RESOURCE_NAME = "task";
const PLURAL_RESOURCE_NAME = "tasks";

export type RetrieveQuery = task.RetrieveSingleParams;

const BASE_QUERY = { includeStatus: true };

/**
 * Parses cached and streamed tasks through the schemas, so a typed hook never hands on
 * the untyped tasks other consumers write into the client cache. A task that fails to
 * parse reads as not cached. Results are memoized per task, as the cache interns its
 * tasks and Flux compares snapshots by identity.
 */
const createCachedParser = <S extends task.Schemas>(schemas: S) => {
  const schema = task.payloadZ(schemas);
  const tasks = new WeakMap<task.Task, task.Task<S> | undefined>();
  const deleted = new WeakMap<query.Deleted<task.Task>, query.Deleted<task.Task<S>>>();
  const parse = (client: Synnax, cached: task.Task): task.Task<S> | undefined => {
    if (tasks.has(cached)) return tasks.get(cached);
    const result = schema.safeParse(cached.payload);
    // payloadZ types each optional schema as a union with its default, so the parsed
    // payload narrows to the typed one here.
    const typed = result.success
      ? client.tasks.sugar(result.data as task.Payload<S>, schemas)
      : undefined;
    tasks.set(cached, typed);
    return typed;
  };
  return (
    client: Synnax,
    cached: query.Cached<task.Task> | undefined,
  ): query.Cached<task.Task<S>> | undefined => {
    if (cached === undefined) return undefined;
    if (!query.Deleted.matches(cached)) return parse(client, cached);
    const held = deleted.get(cached);
    if (held != null) return held;
    const corpse = parse(client, cached.corpse);
    if (corpse === undefined) return undefined;
    const next = new query.Deleted(corpse, cached.deletedAt);
    deleted.set(cached, next);
    return next;
  };
};

export const createRetrieve = <S extends task.Schemas = task.Schemas>(schemas?: S) => {
  const parse = schemas == null ? null : createCachedParser(schemas);
  return Flux.createRetrieve<RetrieveQuery, task.Task<S>>({
    name: RESOURCE_NAME,
    normalizeQuery: (query) => ({ ...BASE_QUERY, ...query }),
    retrieve: async ({ client, query }) =>
      await client.tasks.retrieve({ ...query, schemas }),
    onChange: ({ client, query }, handler) =>
      parse == null
        ? client.tasks.onChange(query, handler as query.ChangeHandler<task.Task>)
        : client.tasks.onChange(query, (cached) => handler(parse(client, cached))),
    getCached: ({ client, query }) =>
      parse == null
        ? (client.tasks.getCached(query) as query.Cached<task.Task<S>> | undefined)
        : parse(client, client.tasks.getCached(query)),
  });
};

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
      const parsed = task.payloadZ(schemas).safeParse(cached.payload);
      if (!parsed.success) return undefined;
      return toFormValues(parsed.data as task.Payload<S>);
    },
    update: async ({ client, ...form }) => {
      const value = form.value();
      // The status is never written back. It belongs to the Driver, and the copy
      // held here is behind the Core whenever a newer one is still in flight, so
      // sending it would overwrite the report of a deploy this save just issued.
      const created = await client.tasks.create(
        {
          key: value.key,
          rack: value.rack,
          name: value.name,
          type: value.type,
          config: value.config,
          snapshot: value.snapshot,
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
            zod.parse(task.statusZ(z.unknown().optional()), result.status, {
              label: "task status",
            }),
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
