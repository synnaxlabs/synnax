// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { linePlot, type workspace } from "@synnaxlabs/client";
import { array } from "@synnaxlabs/x";

import { Flux } from "@/flux";
import { Ontology } from "@/ontology";

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<
  FluxSubStore,
  linePlot.Key,
  linePlot.LinePlot
> = { listeners: [] };

export const FLUX_STORE_KEY = "lineplots";

export interface FluxStore extends Flux.UnaryStore<linePlot.Key, linePlot.LinePlot> {}

export type UseDeleteArgs = linePlot.Params;

interface FluxSubStore extends Flux.Store {
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.ResourceFluxStore;
  [FLUX_STORE_KEY]: FluxStore;
}

export type UseRetrieveArgs = linePlot.SingleRetrieveArgs;

export const retrieveSingle = async ({
  store,
  client,
  params: { key },
}: Flux.RetrieveArgs<UseRetrieveArgs, FluxSubStore>) => {
  const cached = store.lineplots.get(key);
  if (cached != null) return cached;
  const plot = await client.workspaces.lineplots.retrieve({ key });
  return plot;
};

export const { useRetrieve } = Flux.createRetrieve<
  UseRetrieveArgs,
  linePlot.LinePlot,
  FluxSubStore
>({ name: "LinePlot", retrieve: retrieveSingle });

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteArgs, FluxSubStore>({
  name: "LinePlot",
  update: async ({ client, value, rollbacks, store }) => {
    const keys = array.toArray(value);
    const ids = keys.map((k) => linePlot.ontologyID(k));
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.add(store.relationships.delete(relFilter));
    await client.workspaces.lineplots.delete(value);
    return value;
  },
});

export interface UseCreateArgs extends linePlot.New {
  workspace: workspace.Key;
}

export interface UseCreateResult extends linePlot.LinePlot {
  workspace: workspace.Key;
}

export const { useUpdate: useCreate } = Flux.createUpdate<
  UseCreateArgs,
  FluxSubStore,
  UseCreateResult
>({
  name: "LinePlot",
  update: async ({ client, value, store }) => {
    const { workspace, ...rest } = value;
    const l = await client.workspaces.lineplots.create(workspace, rest);
    store.lineplots.set(l.key, l);
    return { ...l, workspace };
  },
});

export interface UseRenameArgs {
  key: linePlot.Key;
  name: string;
}

export const { useUpdate: useRename } = Flux.createUpdate<UseRenameArgs, FluxSubStore>({
  name: "LinePlot",
  update: async ({ client, value, rollbacks, store }) => {
    const { key } = value;
    await client.workspaces.lineplots.rename(key, value.name);
    rollbacks.add(
      store.lineplots.set(key, (p) =>
        p == null ? undefined : { ...p, name: value.name },
      ),
    );
    rollbacks.add(
      store.resources.set(key, (r) =>
        r == null ? undefined : { ...r, name: value.name },
      ),
    );
    return value;
  },
});
