// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  project,
  query,
  type resource,
  type Synnax,
  UnexpectedError,
} from "@synnaxlabs/client";
import { array, deep, type record, verbs } from "@synnaxlabs/x";
import type z from "zod";

import { Flux } from "@/flux";

const RESOURCE_NAME = "project";
const PLURAL_RESOURCE_NAME = "projects";

export type RetrieveQuery = {
  key: project.Key;
};

export const { use } = Flux.createRetrieve<RetrieveQuery, project.Project>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.projects.retrieve(query),
  onChange: ({ client, query }, handler) => client.projects.onChange(query, handler),
  getCached: ({ client, query }) => client.projects.getCached(query),
});

export type ListParams = Pick<
  project.RetrieveRequest,
  "keys" | "offset" | "limit" | "searchTerm"
>;

export const useList = Flux.createList<ListParams, project.Key, project.Project>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.projects.retrieve(query),
  retrieveByKey: async ({ client, key }) => await client.projects.retrieve(key),
  onChange: ({ client, query }, handler) => client.projects.onChange(query, handler),
  getCached: ({ client, query }) => client.projects.getCached(query),
  onChangeByKey: ({ client, key }, handler) => client.projects.onChange(key, handler),
});

export type DeleteParams = project.Key | project.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    const keys = array.toArray(data);
    await client.projects.delete(keys, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export interface RenameParams {
  key: project.Key;
  name: string;
}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, name } = data;
    await client.projects.rename(key, name, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export const formSchema = project.projectZ.partial({ key: true });

export const formPath = deep.path<z.infer<typeof formSchema>>();

const INITIAL_VALUES: z.infer<typeof formSchema> = {
  name: "",
};

export const useForm = Flux.createForm<RetrieveQuery, typeof formSchema>({
  name: RESOURCE_NAME,
  schema: formSchema,
  initialValues: INITIAL_VALUES,
  retrieve: async ({ client, query: { key } }) => await client.projects.retrieve(key),
  getCached: ({ client, query: { key } }) => {
    const cached = client.projects.getCached(key);
    return query.isLive(cached) ? cached : undefined;
  },
  update: async ({ client, value, set }) => {
    const res = await client.projects.create(value());
    set(formPath.key, res.key);
  },
});

interface Owned extends record.KeyedNamed {
  project?: project.Key;
}

type ChildRetriever = (
  client: Synnax,
  request: project.OwnerFilter & { keys?: string[] },
) => Promise<Owned[]>;

/** A resource type that belongs to a project. */
export type ChildType = "schematic" | "lineplot" | "log" | "table" | "panel";

const CHILD_RETRIEVERS: Record<ChildType, ChildRetriever> = {
  schematic: async (client, request) => await client.schematics.retrieve(request),
  lineplot: async (client, request) => await client.lineplots.retrieve(request),
  log: async (client, request) => await client.logs.retrieve(request),
  table: async (client, request) => await client.tables.retrieve(request),
  panel: async (client, request) => await client.panels.retrieve(request),
};

export type RetrieveChildrenQuery = {
  resourceID?: resource.ID;
  types: ChildType[];
};

export interface Child extends record.KeyedNamed {
  type: ChildType;
}

const retrieverOf = (type: resource.Type): ChildRetriever => {
  if (!(type in CHILD_RETRIEVERS))
    throw new UnexpectedError(`a ${type} does not belong to a project`);
  return CHILD_RETRIEVERS[type as ChildType];
};

/**
 * Retrieves the resources of the given types that share the queried resource's
 * project, excluding the resource itself. Returns an empty list when the resource
 * belongs to no project.
 */
export const retrieveChildren = async (
  client: Synnax,
  { resourceID, types }: RetrieveChildrenQuery,
): Promise<Child[]> => {
  if (resourceID == null) return [];
  const [owner] = await retrieverOf(resourceID.type)(client, {
    keys: [resourceID.key],
  });
  if (owner?.project == null) return [];
  const projects = [owner.project];
  const children = await Promise.all(
    types.map(async (type) =>
      (await CHILD_RETRIEVERS[type](client, { projects })).map(({ key, name }) => ({
        key,
        name,
        type,
      })),
    ),
  );
  return children.flat().filter(({ key }) => key !== resourceID.key);
};
