// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type query, rack, task } from "@synnaxlabs/client";
import { array, type optional, verbs } from "@synnaxlabs/x";
import { z } from "zod";

import { Flux } from "@/flux";
import { flux } from "@/flux/aether";

export const RESOURCE_NAME = "task";
const PLURAL_RESOURCE_NAME = "tasks";

export type RetrieveQuery = task.RetrieveSingleParams;

const BASE_QUERY = { includeStatus: true };

/**
 * A typed retrieve parses cached and streamed tasks through the schemas. A task that
 * fails to parse reads as not cached.
 */
export const createRetrieve = <S extends task.Schemas = task.Schemas>(schemas?: S) =>
  Flux.createRetrieve<RetrieveQuery, task.Task<S>>({
    name: RESOURCE_NAME,
    normalizeQuery: (query) => ({ ...BASE_QUERY, ...query }),
    retrieve: async ({ client, query }) =>
      await client.tasks.retrieve({ ...query, schemas }),
    onChange: ({ client, query }, handler) =>
      schemas == null
        ? client.tasks.watch(query).onChange(handler as query.ChangeHandler<task.Task>)
        : client.tasks
            .watch(query)
            .onChange((cached) => handler(client.tasks.parse(cached, schemas))),
    getCached: ({ client, query }) =>
      schemas == null
        ? (client.tasks.watch(query).getCached() as
            query.Cached<task.Task<S>> | undefined)
        : client.tasks.parse(client.tasks.watch(query).getCached(), schemas),
  });

export const { use, useEnsure, useResult, useTombstone, createSelector } =
  createRetrieve();

export interface KeyParams {
  key: task.Key;
}

export const useName = createSelector(({ name }) => name);

export type ListQuery = task.RetrieveMultipleParams;

/**
 * Lists tasks with their statuses, also for a task outside the list. The list leaves
 * out internal tasks unless the query sets `where.internal`.
 */
export const useList = ({
  query,
  ...rest
}: Omit<
  Flux.UseListParams<ListQuery, task.Task>,
  "itemQuery" | "name"
> = {}): Flux.UseListReturn<task.Key, task.Task> =>
  Flux.useList((client) => client.tasks, {
    ...rest,
    name: PLURAL_RESOURCE_NAME,
    query:
      query === null
        ? null
        : {
            ...BASE_QUERY,
            ...query,
            where: { internal: { eq: false }, ...query?.where },
          },
    itemQuery: (key) => ({ ...BASE_QUERY, key }),
  });

export const { use: useMultiple, useResult: useResultMultiple } = Flux.createRetrieve<
  ListQuery,
  task.Task[]
>({
  name: PLURAL_RESOURCE_NAME,
  normalizeQuery: (query) => ({ ...BASE_QUERY, ...query }),
  ...flux.listReads(({ client, query }) => client.tasks.watch(query)),
});

const createFormSchema = <S extends task.Schemas = task.Schemas>(
  schemas: S,
): FormSchema<S> =>
  z.object({
    key: task.keyZ.optional(),
    name: z.string(),
    rack: rack.keyZ,
    type: schemas.type,
    config: schemas.config,
    configHash: z.string(),
    status: task.statusZ(schemas.statusData).optional().nullable(),
  }) as unknown as FormSchema<S>;

export interface FormSchema<S extends task.Schemas = task.Schemas> extends z.ZodType<{
  key?: task.Key;
  name: string;
  rack: rack.Key;
  type: z.infer<S["type"]>;
  config: z.infer<S["config"]>;
  configHash: string;
  status?: task.Status<S["statusData"]>;
}> {}

export interface CreateFormParams<S extends task.Schemas = task.Schemas> {
  schemas: S;
}

export interface InitialValues<
  S extends task.Schemas = task.Schemas,
> extends optional.Optional<
  task.Payload<S>,
  "key" | "rack" | "internal" | "configHash"
> {}

/** Builds the hook for a task's form. */
export const createForm = <S extends task.Schemas = task.Schemas>({
  schemas,
}: CreateFormParams<S>) =>
  Flux.createDocumentForm<task.Key, task.Task, FormSchema<S>>({
    name: RESOURCE_NAME,
    schema: createFormSchema(schemas),
    domain: (client) => client.tasks,
    useEnsure,
    // The cache holds every task untyped. Parsing would cost a full parse on each
    // change, so the form's schema checks each field as it is written instead.
    toValues: ({ payload }) => ({
      key: payload.key,
      name: payload.name,
      rack: payload.rack,
      type: payload.type as z.infer<S["type"]>,
      config: payload.config as z.infer<S["config"]>,
      configHash: payload.configHash ?? "",
      status: payload.status,
    }),
  });

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

export interface UseRenameParams extends Pick<task.Payload, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<UseRenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data }) => {
    const { key, name } = data;
    await client.tasks.rename(key, name);
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

const SEND_VERBS: verbs.Verbs = {
  present: "send",
  participle: "sending",
  past: "sent",
};

export const { useUpdate: useCommand } = Flux.createUpdate<CommandParams>({
  name: "task commands",
  verbs: SEND_VERBS,
  update: async ({ data, client }) => {
    const commands = array.toArray(data);
    const keys = commands.map(({ task }) => task);
    // Warm the cache so command listeners can update the tasks' statuses.
    await client.tasks.retrieve({ key: keys, includeStatus: true });
    await client.tasks.executeCommand({ commands });
    return data;
  },
});
