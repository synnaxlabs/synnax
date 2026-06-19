// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, type ontology, panel } from "@synnaxlabs/client";
import { array } from "@synnaxlabs/x";

import { Flux } from "@/flux";
import { Ontology } from "@/ontology";

export const FLUX_STORE_KEY = "panels";
const RESOURCE_NAME = "panel";
const PLURAL_RESOURCE_NAME = "panels";

export interface FluxStore extends Flux.UndoableUnaryStore<
  panel.Key,
  panel.Panel,
  panel.Action
> {}

export interface FluxSubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.ResourceFluxStore;
}

const kindOfTransaction = (actions: panel.Action[]): string => {
  if (actions.length === 0) return "default";
  // Drag-resize streams ResizeSplit; coalesce them into a single undoable per
  // gesture so one ⌘Z reverses the entire drag.
  if (actions.every((a) => a.type === "resize_split")) return "resize";
  // Same for cross-leaf drags that produce a stream of MoveTab.
  if (actions.every((a) => a.type === "move_tab")) return "move";
  if (actions.length === 1) return actions[0].type;
  return "transaction";
};

const undoableStoreConfig = Flux.createUndoableStore<
  panel.Key,
  panel.Panel,
  panel.Action,
  typeof FLUX_STORE_KEY,
  FluxSubStore
>({
  storeKey: FLUX_STORE_KEY,
  reduce: panel.reduceAll,
  channel: panel.SET_CHANNEL_NAME,
  schema: panel.scopedActionZ,
  kindOf: kindOfTransaction,
});

const DELETE_PANEL_LISTENER: Flux.ChannelListener<FluxSubStore, typeof panel.keyZ> = {
  channel: panel.DELETE_CHANNEL_NAME,
  schema: panel.keyZ,
  onChange: ({ store, changed }) => store.panels.delete(changed),
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<FluxSubStore> = {
  ...undoableStoreConfig,
  listeners: [...undoableStoreConfig.listeners, DELETE_PANEL_LISTENER],
};

export type RetrieveQuery = { key: panel.Key };

const retrieveSingle = async ({
  client,
  query: { key },
  store,
}: Flux.RetrieveParams<RetrieveQuery, FluxSubStore>) => {
  const cached = store.panels.get(key);
  if (cached != null) return cached;
  const p = await client.panels.retrieve(key);
  store.panels.set(p.key, p);
  return p;
};

export const { useRetrieve, useEnsureRetrieved } = Flux.createRetrieve<
  RetrieveQuery,
  panel.Panel,
  FluxSubStore
>({
  name: RESOURCE_NAME,
  retrieve: retrieveSingle,
  mountListeners: ({ store, query: { key }, onChange }) => [
    store.panels.onSet(onChange, key),
  ],
});

export interface TabContent {
  resource?: ontology.ID;
  view?: panel.TabView;
}

// tabContent flattens a tab's variant into the optional content pair consumed by
// render props, so consumers can read resource/view without switching on variant.
export const tabContent = (tab: panel.Tab): TabContent => {
  switch (tab.variant) {
    case "resource":
      return { resource: tab.resource };
    case "view":
      return { view: tab };
    case "empty":
      return {};
  }
};

export interface SelectKeyArgs {
  key: panel.Key;
}

const requirePanel = (store: FluxSubStore, key: panel.Key): panel.Panel => {
  const p = store.panels.get(key);
  if (p == null) throw new NotFoundError(`Panel with key ${key} not found`);
  return p;
};

// useSelectRoot selects the panel's stored tree root. The reference only
// changes when the document changes, so consumers can memoize derivations on
// it directly.
export const useSelectRoot = Flux.createSelector<
  FluxSubStore,
  SelectKeyArgs,
  panel.Node
>({
  subscribe: (store, { key }, notify) => store.panels.onSet(notify, key),
  select: (store, { key }) => requirePanel(store, key).root,
});

export interface SelectTabContentArgs {
  key: panel.Key;
  tabKey: string;
}

export const useSelectTab = Flux.createSelector<
  FluxSubStore,
  SelectTabContentArgs,
  panel.Tab
>({
  subscribe: (store, { key }, notify) => store.panels.onSet(notify, key),
  select: (store, { key, tabKey }) => {
    const tab = panel.findTab(requirePanel(store, key).root, tabKey);
    if (tab == null)
      throw new NotFoundError(`Tab with key ${tabKey} not found in panel ${key}`);
    return tab;
  },
});

export interface ListParams extends Pick<panel.RetrieveRequest, "offset" | "limit"> {}

export const useList = Flux.createList<
  ListParams,
  panel.Key,
  panel.Panel,
  FluxSubStore
>({
  name: PLURAL_RESOURCE_NAME,
  retrieveCached: ({ store }) => store.panels.list(),
  retrieve: async ({ client, query }) => await client.panels.retrieve(query),
  retrieveByKey: async ({ key, ...rest }) =>
    await retrieveSingle({ ...rest, query: { key } }),
  mountListeners: ({ store, onChange, onDelete }) => [
    store.panels.onSet(onChange),
    store.panels.onDelete(onDelete),
  ],
});

export interface CreateParams extends panel.New {}

export const { useUpdate: useCreate } = Flux.createUpdate<
  CreateParams,
  FluxSubStore,
  panel.Panel
>({
  name: RESOURCE_NAME,
  verbs: Flux.CREATE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const optimistic = panel.panelZ.parse(data);
    rollbacks.push(store.panels.set(optimistic));
    const created = await client.panels.create(optimistic);
    store.panels.set(created);
    return created;
  },
});

export interface RenameParams extends Pick<panel.Panel, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data, rollbacks, store }) => {
    const { key, name } = data;
    rollbacks.push(Flux.partialUpdate(store.panels, key, { name }));
    rollbacks.push(Ontology.renameFluxResource(store, panel.ontologyID(key), name));
    await client.panels.rename(key, name);
    return data;
  },
});

export type DeleteParams = panel.Key | panel.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const keys = array.toArray(data);
    const ids = panel.ontologyID(keys);
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.push(store.relationships.delete(relFilter));
    rollbacks.push(store.resources.delete(keys));
    rollbacks.push(store.panels.delete(keys));
    await client.panels.delete(keys);
    return data;
  },
});

export const { useDispatch, useUndo, useRedo } = Flux.createDispatch<
  panel.Key,
  panel.Panel,
  panel.Action,
  typeof FLUX_STORE_KEY,
  FluxSubStore
>({
  storeKey: FLUX_STORE_KEY,
  send: ({ client, key, actions, dispatchKey }) =>
    client.panels.dispatch(key, dispatchKey, actions),
});
