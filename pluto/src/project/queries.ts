// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, project } from "@synnaxlabs/client";
import { array, type record, verbs } from "@synnaxlabs/x";
import type z from "zod";

import { Flux } from "@/flux";

const RESOURCE_NAME = "project";
const PLURAL_RESOURCE_NAME = "projects";

export type RetrieveQuery = {
  key: project.Key;
};

export const { useRetrieve } = Flux.createRetrieve<RetrieveQuery, project.Project>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.projects.retrieve(query),
  subscribe: ({ client, query }, handler) => client.projects.onChange(query, handler),
  getCached: ({ client, query }) => client.projects.getCached(query),
});

export type ListParams = Pick<project.RetrieveRequest, "keys" | "offset" | "limit">;

export const useList = Flux.createList<ListParams, project.Key, project.Project>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.projects.retrieve(query),
  retrieveByKey: async ({ client, key }) => await client.projects.retrieve(key),
  subscribe: ({ client, query }, handler) => client.projects.onChange(query, handler),
  getCached: ({ client, query }) => client.projects.getCached(query),
  subscribeByKey: ({ client, key }, handler) => client.projects.onChange(key, handler),
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
  update: async ({ client, data }) => {
    const { key, name } = data;
    await client.projects.rename(key, name);
    return data;
  },
});

export type RetrieveGroupQuery = Record<string, never>;

export const { useRetrieve: useRetrieveGroupID } = Flux.createRetrieve<
  RetrieveGroupQuery,
  ontology.ID | undefined
>({
  name: "Project Group",
  retrieve: async ({ client }) => {
    const res = await client.ontology.children.retrieve({ ids: ontology.ROOT_ID });
    return res.find((r) => r.name === "Projects")?.id;
  },
});

export const formSchema = project.projectZ.partial({ key: true });

const INITIAL_VALUES: z.infer<typeof formSchema> = {
  name: "",
  layout: {},
};

export const useForm = Flux.createForm<Partial<RetrieveQuery>, typeof formSchema>({
  name: RESOURCE_NAME,
  schema: formSchema,
  initialValues: INITIAL_VALUES,
  retrieve: async ({ client, query: { key }, reset }) => {
    if (key == null) return;
    reset(await client.projects.retrieve(key));
  },
  update: async ({ client, value, set }) => {
    const res = await client.projects.create(value());
    set("key", res.key);
  },
});

export interface SaveLayoutParams extends project.SetLayoutParams {}

const LAYOUT_RESOURCE_NAME = "project layout";

export const { useUpdate: useSaveLayout } = Flux.createUpdate<SaveLayoutParams>({
  name: LAYOUT_RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, layout } = data;
    await client.projects.setLayout(key, layout, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export type RetrieveChildrenQuery = {
  resourceID?: ontology.ID;
  types: ontology.ResourceType[];
};

const collectChildren = async (
  client: Flux.RetrieveParams<RetrieveChildrenQuery>["client"],
  parentID: ontology.ID,
  types: ontology.ResourceType[],
  exclude?: string,
): Promise<record.KeyedNamed[]> => {
  const children = await client.ontology.children.retrieve({
    ids: parentID,
    types: [...types, "group"],
  });
  const results: record.KeyedNamed[] = [];
  for (const child of children)
    if (types.includes(child.id.type) && child.id.key !== exclude)
      results.push({ key: child.id.key, name: child.name });
    else if (child.id.type === "group")
      results.push(...(await collectChildren(client, child.id, types, exclude)));
  return results;
};

const findProjectAncestor = async (
  client: Flux.RetrieveParams<RetrieveChildrenQuery>["client"],
  resourceID: ontology.ID,
): Promise<ontology.ID | null> => {
  const parents = await client.ontology.parents.retrieve({ ids: resourceID });
  for (const parent of parents) {
    if (parent.id.type === "project") return parent.id;
    if (parent.id.type === "group") return await findProjectAncestor(client, parent.id);
  }
  return null;
};

const retrieveChildrenImpl = async ({
  client,
  query: { resourceID, types },
}: Flux.RetrieveParams<RetrieveChildrenQuery>): Promise<record.KeyedNamed[]> => {
  if (resourceID == null) return [];
  const projectID = await findProjectAncestor(client, resourceID);
  if (projectID == null) return [];
  return await collectChildren(client, projectID, types, resourceID.key);
};

export const { useRetrieve: useRetrieveChildren } = Flux.createRetrieve<
  RetrieveChildrenQuery,
  record.KeyedNamed[]
>({
  name: "project children",
  retrieve: retrieveChildrenImpl,
});
