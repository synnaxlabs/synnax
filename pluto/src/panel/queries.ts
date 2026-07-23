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
  ontology,
  panel,
  project,
  UnexpectedError,
} from "@synnaxlabs/client";
import { array, compare, deep, type optional, type record } from "@synnaxlabs/x";
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
    retrieveCached: ({ store, query: { key } }) => store.panels.get(key),
    mountListeners: ({ store, query: { key }, onChange }) => [
      store.panels.onSet(onChange, key),
    ],
  });

export type RetrieveByProjectQuery = { project: project.Key };

const isPanelChildOf = (
  relationship: ontology.Relationship,
  projectKey: project.Key,
): boolean =>
  ontology.matchRelationship(relationship, {
    type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
    from: project.ontologyID(projectKey),
  }) && relationship.to.type === panel.TYPE_ONTOLOGY_ID.type;

const replaceOrAppend = (panels: panel.Panel[], next: panel.Panel): panel.Panel[] => {
  const i = panels.findIndex(({ key }) => key === next.key);
  if (i === -1) return [...panels, next];
  const out = [...panels];
  out[i] = next;
  return out;
};

// A panel's parent lives in the ontology graph and is absent from the panel record, so
// membership is resolved through the project's children.
export const { useRetrieve: useRetrieveByProject } = Flux.createRetrieve<
  RetrieveByProjectQuery,
  panel.Panel[],
  FluxSubStore
>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query: { project: projectKey }, store }) => {
    const children = await client.ontology.retrieveChildren(
      project.ontologyID(projectKey),
    );
    const keys = children
      .filter(({ id }) => id.type === panel.TYPE_ONTOLOGY_ID.type)
      .map(({ id }) => id.key);
    if (keys.length === 0) return [];
    const panels = await client.panels.retrieve(keys);
    panels.forEach((p) => store.panels.set(p.key, p));
    return panels;
  },
  mountListeners: ({ store, client, query: { project: projectKey }, onChange }) => [
    store.relationships.onSet(async (relationship) => {
      if (!isPanelChildOf(relationship, projectKey)) return;
      const p = await client.panels.retrieve(relationship.to.key);
      onChange((prev) => replaceOrAppend(prev ?? [], p));
    }),
    store.relationships.onDelete((changed) => {
      const relationship = ontology.relationshipZ.parse(changed);
      if (!isPanelChildOf(relationship, projectKey)) return;
      onChange((prev) => prev?.filter(({ key }) => key !== relationship.to.key));
    }),
    // Only updates panels already in the project; a panel absent here belongs to
    // another project and must not be pulled in.
    store.panels.onSet((p) =>
      onChange((prev) =>
        prev?.some(({ key }) => key === p.key) ? replaceOrAppend(prev, p) : prev,
      ),
    ),
    store.panels.onDelete((key) =>
      onChange((prev) => prev?.filter((p) => p.key !== key)),
    ),
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
        get({
          ...args,
          key: Scope.require(args?.key ?? key),
          tabKey: TabScope.require(args?.tabKey ?? tabKey),
        } as Args),
      [get, key, tabKey],
    );
  };
  return [boundUseSelect, boundUseGet];
};

export interface SelectNodeArgs extends SelectKeyArgs {
  nodeKey: number;
}

// useSelectNodeVariant selects only the variant of the node at the given path, so a
// component that branches on split-vs-leaf does not re-render on structure changes
// within the same variant.
export const [useSelectNodeVariant, useGetNodeVariant] = Scope.bindSelector(
  Flux.createSelector<FluxSubStore, SelectNodeArgs, panel.Node["variant"]>({
    subscribe: (store, { key }, notify) => store.panels.onSet(notify, key),
    select: (store, { key, nodeKey }) => {
      const node = panel.findNode(requirePanel(store, key).root, nodeKey);
      if (node == null) throw new NotFoundError(`Node at path ${nodeKey} not found`);
      return node.variant;
    },
  }),
);

// useSelectLeafNode selects the leaf node at the given path, including its tab keys.
export const [useSelectLeafNode, useGetLeafNode] = Scope.bindSelector(
  Flux.createSelector<
    FluxSubStore,
    SelectNodeArgs,
    Omit<panel.NodeLeaf, "tabs"> & { tabs: panel.TabKey[] },
    panel.Node
  >({
    subscribe: (store, { key }, notify) => store.panels.onSet(notify, key),
    select: (store, { key, nodeKey }) => {
      const node = panel.findNode(requirePanel(store, key).root, nodeKey);
      if (node == null) throw new NotFoundError(`Node at path ${nodeKey} not found`);
      return node;
    },
    transform: (node) => {
      if (node.variant !== "leaf") throw new UnexpectedError("node is not a leaf");
      return { ...node, tabs: node.tabs.map((t) => t.key) };
    },
    equal: deep.equal,
  }),
);

// useSelectSplitNode selects the split node at the given path, including its direction
// and size.
export const [useSelectSplitNode, useGetSplitNode] = Scope.bindSelector(
  Flux.createSelector<FluxSubStore, SelectNodeArgs, panel.NodeSplit, panel.Node>({
    subscribe: (store, { key }, notify) => store.panels.onSet(notify, key),
    select: (store, { key, nodeKey }) => {
      const node = panel.findNode(requirePanel(store, key).root, nodeKey);
      if (node == null) throw new NotFoundError(`Node at path ${nodeKey} not found`);
      return node;
    },
    transform: (node) => {
      if (node.variant !== "split") throw new UnexpectedError("node is not a split");
      return node;
    },
    equal: deep.equal,
  }),
);

const tabKeys = (store: FluxSubStore, key: panel.Key): string[] => {
  const p = store.panels.get(key);
  if (p == null) return [];
  const tabKeys: string[] = [];
  const visit = (node: panel.Node | undefined) => {
    if (node == null) return;
    if (node.variant === "split") {
      visit(node.first);
      visit(node.last);
    } else tabKeys.push(...node.tabs.flatMap((t) => t.key));
  };
  visit(p.root);
  return tabKeys.sort();
};

// useSelectLeafTabGroups selects each leaf's tab keys, deep-equal compared so the mosaic
// root re-renders only when tab membership changes — not on a resize or a content change.
export const [useSelectTabKeys, useGetTabKeys] = Scope.bindSelector(
  Flux.createSelector<FluxSubStore, SelectKeyArgs, string[]>({
    subscribe: (store, { key }, notify) => store.panels.onSet(notify, key),
    select: (store, { key }) => tabKeys(store, key),
    equal: compare.arraysEqual,
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

// leafTabGroups returns each leaf's ordered tab keys, or null when the panel is not
// cached.
const leafTabGroups = (
  store: FluxSubStore,
  key: panel.Key,
): panel.TabKey[][] | null => {
  const p = store.panels.get(key);
  if (p == null) return null;
  const groups: panel.TabKey[][] = [];
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

export interface SelectSelectionArgs {
  key?: panel.Key;
  selected: panel.TabKey[];
}

const NOOP = () => {};

const [useSelectSelectionBase, useGetSelectionBase] = Flux.createSelector<
  FluxSubStore,
  SelectSelectionArgs,
  panel.TabKey[],
  panel.TabKey[][] | null
>({
  subscribe: (store, { key }, notify) =>
    key == null ? NOOP : store.panels.onSet(notify, key),
  select: (store, { key }) => (key == null ? null : leafTabGroups(store, key)),
  transform: (groups, { selected }) => {
    if (groups == null) return selected;
    const leafOf = new Map<panel.TabKey, number>();
    groups.forEach((tabs, leaf) => tabs.forEach((t) => leafOf.set(t, leaf)));
    const claimed = new Set<number>();
    const out: panel.TabKey[] = [];
    for (const tabKey of selected) {
      const leaf = leafOf.get(tabKey);
      if (leaf == null || claimed.has(leaf)) continue;
      claimed.add(leaf);
      out.push(tabKey);
    }
    groups.forEach((tabs, leaf) => {
      if (!claimed.has(leaf) && tabs.length > 0) out.push(tabs[0]);
    });
    return out;
  },
  equal: compare.arraysEqual,
});

/**
 * useSelectSelection resolves a recency-ordered list of selected tab keys into the
 * panel's exact selection: one tab per leaf, most recently selected first. Keys no
 * longer in the tree are dropped, at most one key is kept per leaf, and a leaf none
 * of whose tabs are selected contributes its first tab at the end of the list. The
 * first key is the panel's focused tab. The key defaults to the surrounding Panel
 * scope; when neither is present, or the panel is not cached, the list is returned
 * unresolved.
 */
export const useSelectSelection = (args: SelectSelectionArgs): panel.TabKey[] => {
  const key = Scope.useOptional(args.key);
  return useSelectSelectionBase({ ...args, key });
};

/** useGetSelection returns a getter reading {@link useSelectSelection} on demand. */
export const useGetSelection = (): ((args: SelectSelectionArgs) => panel.TabKey[]) => {
  const scoped = Scope.useOptional();
  const get = useGetSelectionBase();
  return useCallback(
    (args: SelectSelectionArgs) => get({ ...args, key: args.key ?? scoped }),
    [get, scoped],
  );
};

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
// Resource renderers call this to learn their own document key. Throws
// UnexpectedError when the active tab is not a resource tab: only renderers mounted
// for a resource tab may call this, so a wrong-variant read is a programmer bug.
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
// compared so it re-renders only when the args contents actually change. Throws
// UnexpectedError when the active tab is not a view tab (see useSelectTabResource).
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

// useCloseDeletedResourceTabs removes any resource-backed tab whose resource has
// been deleted, across every cached panel. A resource left as a tombstone would
// render "Not found" indefinitely, so its tab is closed with the resource.
// removeTab on an absent tab is a no-op, so redundant firings across windows are
// harmless.
export const useCloseDeletedResourceTabs = (): void => {
  const store = Flux.useStore<FluxSubStore>();
  const { dispatch } = useDispatch();
  Ontology.useResourceDeleteSynchronizer(
    useCallback(
      (id) =>
        store.panels.list().forEach((p) => {
          const tab = panel.findTabByResource(p.root, id);
          if (tab != null)
            dispatch({ key: p.key, actions: panel.removeTab({ key: tab.key }) });
        }),
      [store, dispatch],
    ),
  );
};

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
