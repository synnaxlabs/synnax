import { group, type ontology } from "@synnaxlabs/client";

import { Flux } from "@/flux";

export interface CreateParams {
  key?: group.Key;
}

export interface CreateValue extends group.Payload {
  parent: ontology.ID;
}

export const create = Flux.createUpdate<CreateParams, CreateValue>({
  name: "Group",
  update: async ({ value, client, onChange }) => {
    const { parent } = value;
    const res = await client.ontology.groups.create(parent, value.name, value.key);
    onChange({ ...res, parent });
  },
});

export interface ListParams {
  parent?: ontology.ID;
  searchTerm?: string;
  offset?: number;
  limit?: number;
}

export const useList = Flux.createList<ListParams, group.Key, group.Payload>({
  name: "Group",
  retrieve: async ({ client, params }) => {
    if (params.parent == null) return [];
    const res = await client.ontology.retrieveChildren(params.parent, {
      ...params,
      types: ["group"],
    });
    return res.map((r) => group.groupZ.parse(r.data));
  },
  retrieveByKey: async ({ client, key }) => {
    const res = await client.ontology.retrieve(group.ontologyID(key));
    return group.groupZ.parse(res.data);
  },
});

export interface RenameParams {
  key: string;
}

export const useRename = Flux.createUpdate<RenameParams, string>({
  name: "Group",
  update: async ({ client, value, params }) =>
    await client.ontology.groups.rename(params.key, value),
}).useDirect;

export interface DeleteParams {
  key: string;
}

export const useDelete = Flux.createUpdate<DeleteParams, void>({
  name: "Group",
  update: async ({ client, params }) => await client.ontology.groups.delete(params.key),
}).useDirect;
