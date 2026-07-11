// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  NotFoundError,
  type ontology,
  panel,
  UnexpectedError,
} from "@synnaxlabs/client";
import { array, deep, type optional, type record } from "@synnaxlabs/x";
import { useCallback, useMemo } from "react";
import { type z } from "zod";

import { Flux } from "@/flux";
import { Ontology } from "@/ontology";
import { Scope, TabScope } from "@/panel/scope";

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

export const { useRetrieve, useEnsureRetrieved, useRetrieveEffect } =
  Flux.createRetrieve<RetrieveQuery, panel.Panel, FluxSubStore>({
    name: RESOURCE_NAME,
    retrieve: retrieveSingle,
    mountListeners: ({ store, query: { key }, onChange }) => [
      store.panels.onSet(onChange, key),
    ],
  });

export interface SelectKeyArgs {
  key: panel.Key;
}

const requirePanel = (store: FluxSubStore, key: panel.Key): panel.Panel => {
  const p = store.panels.get(key);
  if (p == null) throw new NotFoundError(`Panel with key ${key} not found`);
  return p;
};

export interface SelectTabContentArgs {
  key: panel.Key;
  tabKey: panel.TabKey;
}

const selectRequiredTab = (
  store: FluxSubStore,
  { key, tabKey }: SelectTabContentArgs,
): panel.Tab => {
  const tab = panel.findTab(requirePanel(store, key).root, tabKey);
  if (tab == null)
    throw new NotFoundError(`Tab with key ${tabKey} not found in panel ${key}`);
  return tab;
};

// bindTabHook lifts a hook needing both a panel key and a tab key into one whose keys are
// sourced from the surrounding Panel and Tab scopes; either may be overridden explicitly.
// The two-level analogue of scope.bindHook.
type BoundTabHook<Args extends SelectTabContentArgs, R> = optional.Arg<
  optional.Optional<Args, "key" | "tabKey">,
  R
>;

const bindTabHook =
  <Args extends SelectTabContentArgs, R>(
    hook: (args: Args) => R,
  ): BoundTabHook<Args, R> =>
  (args?: optional.Optional<Args, "key" | "tabKey">): R => {
    const key = Scope.use(args?.key);
    const tabKey = TabScope.use(args?.tabKey);
    return hook({ ...args, key, tabKey } as Args);
  };

// bindTabSelector is the two-level analogue of scope.bindSelector: it binds a
// createSelector pair to the Panel and Tab scopes. The reactive hook resolves both keys
// from scope; the getter it returns injects both (overridable per call) into each read.
const bindTabSelector = <Args extends SelectTabContentArgs, Selected>([
  useSelect,
  useGet,
]: Flux.Selector<Args, Selected>): [
  BoundTabHook<Args, Selected>,
  () => optional.Arg<optional.Optional<Args, "key" | "tabKey">, Selected>,
] => {
  const boundUseSelect = bindTabHook(useSelect);
  const boundUseGet = (): optional.Arg<
    optional.Optional<Args, "key" | "tabKey">,
    Selected
  > => {
    const key = Scope.useOptional();
    const tabKey = TabScope.useOptional();
    const get = useGet();
    return useCallback(
      (args?: optional.Optional<Args, "key" | "tabKey">) =>
        get({ key, tabKey, ...args } as Args),
      [get, key, tabKey],
    );
  };
  return [boundUseSelect, boundUseGet];
};

export interface NodeStructure {
  /** variant discriminates a split node from a leaf node. */
  variant: panel.Node["variant"];
  /** tabs holds the ordered tab keys of a leaf node. */
  tabs?: string[];
  /** direction is the split axis of a split node. */
  direction?: Extract<panel.Node, { variant: "split" }>["direction"];
  /** size is the split ratio of a split node as a decimal in [0, 1]. */
  size?: number;
}

const structureOf = (node: panel.Node | null): NodeStructure | null => {
  if (node == null) return null;
  if (node.variant === "split")
    return { variant: "split", direction: node.direction, size: node.size };
  return { variant: "leaf", tabs: node.tabs.map((t) => t.key) };
};

export interface SelectNodeArgs extends SelectKeyArgs {
  path: number;
}

// useSelectNode selects the structural descriptor of the single node at the given path: a
// split's direction and size, or a leaf's ordered tab keys. It carries no tab content and
// is deep-equal compared, so a node re-renders only when its own structure changes — a
// resize touches one split, a tab insert one leaf, and a tab's content change nothing.
export const [useSelectNode, useGetNode] = Scope.bindSelector(
  Flux.createSelector<
    FluxSubStore,
    SelectNodeArgs,
    NodeStructure | null,
    panel.Node | null
  >({
    subscribe: (store, { key }, notify) => store.panels.onSet(notify, key),
    select: (store, { key, path }) =>
      panel.walkPath(requirePanel(store, key).root, path),
    transform: structureOf,
    equal: deep.equal,
  }),
);

// leafTabGroups returns each leaf's ordered tab keys, in traversal order. It is the
// whole-tree input the mosaic root needs to enumerate stable portal nodes and resolve the
// visible tab per leaf, without re-deriving structure inside the render path.
export const leafTabGroups = (store: FluxSubStore, key: panel.Key): string[][] => {
  const p = store.panels.get(key);
  if (p == null) return [];
  const groups: string[][] = [];
  const visit = (node: panel.Node | undefined) => {
    if (node == null) return;
    if (node.variant === "split") {
      visit(node.first);
      visit(node.last);
    } else groups.push(node.tabs.map((t) => t.key));
  };
  visit(p.root);
  return groups;
};

// useSelectLeafTabGroups selects each leaf's tab keys, deep-equal compared so the mosaic
// root re-renders only when tab membership changes — not on a resize or a content change.
export const [useSelectLeafTabGroups, useGetLeafTabGroups] = Scope.bindSelector(
  Flux.createSelector<FluxSubStore, SelectKeyArgs, string[][]>({
    subscribe: (store, { key }, notify) => store.panels.onSet(notify, key),
    select: (store, { key }) => leafTabGroups(store, key),
    equal: deep.equal,
  }),
);

// useSelectRoot selects the panel's raw stored tree root.
export const [useSelectRoot, useGetRoot] = Scope.bindSelector(
  Flux.createSelector<FluxSubStore, SelectKeyArgs, panel.Node>({
    subscribe: (store, { key }, notify) => store.panels.onSet(notify, key),
    select: (store, { key }) => requirePanel(store, key).root,
  }),
);

// useSelectRoot selects the panel's raw stored tree root.
export const [useSelectName, useGetName] = Scope.bindSelector(
  Flux.createSelector<FluxSubStore, SelectKeyArgs, string>({
    subscribe: (store, { key }, notify) => store.panels.onSet(notify, key),
    select: (store, { key }) => requirePanel(store, key).name,
  }),
);

export const allTabKeys = (store: FluxSubStore, key: panel.Key): panel.TabKey[] => {
  const p = store.panels.get(key);
  if (p == null) return [];
  const keys: panel.TabKey[] = [];
  const visit = (node: panel.Node | undefined) => {
    if (node == null) return;
    if (node.variant === "split") {
      visit(node.first);
      visit(node.last);
    } else node.tabs.forEach((t) => keys.push(t.key));
  };
  visit(p.root);
  return keys;
};

export interface SelectFilteredTabKeysArgs extends SelectKeyArgs {
  tabKeys: panel.TabKey[];
}

export const [useSelectFilteredTabKeys, useGetFilteredTabKeys] = Scope.bindSelector(
  Flux.createSelector<
    FluxSubStore,
    SelectFilteredTabKeysArgs,
    panel.TabKey[],
    panel.TabKey[]
  >({
    subscribe: (store, { key }, notify) => store.panels.onSet(notify, key),
    select: (store, { key }) => allTabKeys(store, key),
    transform: (existingKeys, { tabKeys }) => {
      const existingSet = new Set(existingKeys);
      return tabKeys.filter((k) => existingSet.has(k));
    },
    equal: deep.equal,
  }),
);

export interface SelectFilterTabKeyArgs extends SelectKeyArgs {
  tabKey?: panel.TabKey;
}

export const [useSelectFilterTabKey, useGetFilterTabKey] = Scope.bindSelector(
  Flux.createSelector<FluxSubStore, SelectFilterTabKeyArgs, panel.TabKey | undefined>({
    subscribe: (store, { key }, notify) => store.panels.onSet(notify, key),
    select: (store, { key, tabKey }) => {
      if (tabKey == null) return undefined;
      const p = store.panels.get(key);
      if (p == null) return undefined;
      return panel.findTab(p.root, tabKey) == null ? undefined : tabKey;
    },
  }),
);

export const [useSelectTab, useGetTab] = bindTabSelector(
  Flux.createSelector<FluxSubStore, SelectTabContentArgs, panel.Tab>({
    subscribe: (store, { key }, notify) => store.panels.onSet(notify, key),
    select: (store, args) => selectRequiredTab(store, args),
  }),
);

const selectRequiredTabLeaf = (
  store: FluxSubStore,
  { key, tabKey }: SelectTabContentArgs,
): panel.NodeLeaf => {
  const leaf = panel.findTabLeaf(requirePanel(store, key).root, tabKey);
  if (leaf == null)
    throw new NotFoundError(`Leaf holding tab ${tabKey} not found in panel ${key}`);
  return leaf;
};

// useSelectTabLeaf selects the leaf node holding the active tab. The leaf is a live
// reference into the stored tree, so immer's structural sharing gives it stable
// identity across dispatches that don't touch it.
export const [useSelectTabLeaf, useGetTabLeaf] = bindTabSelector(
  Flux.createSelector<FluxSubStore, SelectTabContentArgs, panel.NodeLeaf>({
    subscribe: (store, { key }, notify) => store.panels.onSet(notify, key),
    select: (store, args) => selectRequiredTabLeaf(store, args),
  }),
);

// useSelectTabVariant selects only the content variant of the active tab, so a
// component that branches on resource-vs-view does not re-render on content edits
// within the same variant.
export const [useSelectTabVariant, useGetTabVariant] = bindTabSelector(
  Flux.createSelector<FluxSubStore, SelectTabContentArgs, panel.TabType>({
    subscribe: (store, { key }, notify) => store.panels.onSet(notify, key),
    select: (store, args) => selectRequiredTab(store, args).variant,
  }),
);

// useSelectTabType selects the renderer type of the active tab: the resource's
// ontology type for resource tabs, the view type for view tabs. Components that
// render by type do not re-render when a view's args change.
export const [useSelectTabType, useGetTabType] = bindTabSelector(
  Flux.createSelector<FluxSubStore, SelectTabContentArgs, string>({
    subscribe: (store, { key }, notify) => store.panels.onSet(notify, key),
    select: (store, args) => {
      const tab = selectRequiredTab(store, args);
      return tab.variant === "resource" ? tab.resource.type : tab.type;
    },
  }),
);

export interface SelectMaybeTabTypeArgs {
  key?: panel.Key | null;
  tabKey?: panel.TabKey | null;
}

// useSelectTabResource selects the ontology ID displayed by the active resource tab.
// Resource renderers call this to learn their own document key. Returns null when
// the active tab is not a resource tab: mounted hooks recompute during a live
// view-to-resource swap before the outgoing renderer unmounts, so a wrong-variant
// read must degrade rather than throw.
export const [useSelectTabResource, useGetTabResource] = bindTabSelector(
  Flux.createSelector<FluxSubStore, SelectTabContentArgs, ontology.ID>({
    subscribe: (store, { key }, notify) => store.panels.onSet(notify, key),
    select: (store, args) => {
      const tab = selectRequiredTab(store, args);
      if (tab.variant !== "resource")
        throw new UnexpectedError(
          `attempted to select resource on view tab ${tab.key}`,
        );
      return tab.resource;
    },
    equal: deep.equal,
  }),
);

// useSelectTabArgs selects only the opaque args of the active view tab, deep-equal
// compared so it re-renders only when the args contents actually change. Returns
// null when the active tab is not a view tab (see useSelectTabResource for why
// wrong-variant reads must not throw).
export const [useSelectTabArgs, useGetTabArgs] = bindTabSelector(
  Flux.createSelector<FluxSubStore, SelectTabContentArgs, record.Unknown>({
    subscribe: (store, { key }, notify) => store.panels.onSet(notify, key),
    select: (store, args) => {
      const tab = selectRequiredTab(store, args);
      if (tab.variant !== "view")
        throw new UnexpectedError(
          `attempted to select args on resource tab ${tab.key}`,
        );
      return tab.args;
    },
    equal: deep.equal,
  }),
);

// createSelectTabArgs builds a tab-scoped hook that selects the active view tab's
// args and parses them with the given schema, for a typed view of a known content
// kind. Returns null when the active tab is not a view tab.
export const createSelectTabArgs =
  <Z extends z.ZodType>(schema: Z): (() => z.output<Z>) =>
  () => {
    const args = useSelectTabArgs({});
    return useMemo(() => schema.parse(args), [args, schema]);
  };

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
  update: async ({ client, data, store, rollbacks, onOptimisticComplete }) => {
    const optimistic = panel.panelZ.parse(data);
    rollbacks.push(store.panels.set(optimistic));
    await onOptimisticComplete(optimistic);
    // onOptimisticComplete may have dispatched further local mutations against this
    // same key (e.g. inserting a tab) before the panel exists on the cluster. Send the
    // latest cached doc rather than the pre-mutation snapshot, or the server response
    // below would stomp those local changes back out.
    const latest = store.panels.get(optimistic.key) ?? optimistic;
    const created = await client.panels.create(latest);
    store.panels.set(created);
    return created;
  },
});

export interface RenameParams extends Pick<panel.Panel, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data, rollbacks, store, onOptimisticComplete }) => {
    const { key, name } = data;
    rollbacks.push(Flux.partialUpdate(store.panels, key, { name }));
    rollbacks.push(Ontology.renameFluxResource(store, panel.ontologyID(key), name));
    await onOptimisticComplete(data);
    await client.panels.rename(key, name);
    return data;
  },
});

export type DeleteParams = panel.Key | panel.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, store, rollbacks, onOptimisticComplete }) => {
    const keys = array.toArray(data);
    const ids = panel.ontologyID(keys);
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.push(store.relationships.delete(relFilter));
    rollbacks.push(store.resources.delete(keys));
    rollbacks.push(store.panels.delete(keys));
    await onOptimisticComplete(data);
    await client.panels.delete(keys);
    return data;
  },
});

export const {
  useDispatch,
  useSingleDispatch: useSingleDispatchBase,
  useUndo: useUndoBase,
  useRedo: useRedoBase,
} = Flux.createDispatch<
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

export const useSingleDispatch = Scope.bindHook(useSingleDispatchBase);
export const useUndo = Scope.bindHook(useUndoBase);
export const useRedo = Scope.bindHook(useRedoBase);

// useSetCurrentTabResource swaps the current tab's content to the given resource,
// clearing any view. The selector flow uses this to fill the tab in place once the
// user picks a visualization.
export const useSetCurrentTabResource = (): ((resource: ontology.ID) => void) => {
  const dispatch = useSingleDispatch();
  const tabKey = TabScope.use();
  return useCallback(
    (resource: ontology.ID) =>
      dispatch(panel.setTabResource({ key: tabKey, resource })),
    [dispatch, tabKey],
  );
};

// useSetCurrentTabView swaps the current tab's content to the given inline view,
// clearing any resource.
export const useSetCurrentTabView = (): ((view: panel.View) => void) => {
  const dispatch = useSingleDispatch();
  const tabKey = TabScope.use();
  return useCallback(
    (view: panel.View) => dispatch(panel.setTabView({ key: tabKey, view })),
    [dispatch, tabKey],
  );
};
