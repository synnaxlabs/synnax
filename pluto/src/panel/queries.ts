// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  cache,
  NotFoundError,
  type ontology,
  panel,
  type Synnax as Client,
  UnexpectedError,
} from "@synnaxlabs/client";
import { array, compare, deep, type optional, type record } from "@synnaxlabs/x";
import { useCallback, useMemo } from "react";
import { type z } from "zod";

import { Flux } from "@/flux";
import { Scope, TabScope } from "@/panel/scope";

const RESOURCE_NAME = "panel";
const PLURAL_RESOURCE_NAME = "panels";

export type RetrieveQuery = { key: panel.Key };

export const { useRetrieve, useEnsureRetrieved, useRetrieveEffect } =
  Flux.createRetrieve<RetrieveQuery, panel.Panel>({
    name: RESOURCE_NAME,
    retrieve: async ({ client, query: { key } }) => await client.panels.retrieve(key),
    subscribe: ({ client, query: { key } }, handler) =>
      client.panels.onChange(key, handler),
    getCached: ({ client, query: { key } }) => client.panels.getCached(key),
  });

export interface SelectKeyParams {
  key: panel.Key;
}

const requirePanel = (client: Client | null, key: panel.Key): panel.Panel => {
  const cached = client?.panels.getCached(key);
  if (cached == null || cached.variant === "deleted")
    throw new NotFoundError(`Panel with key ${key} not found`);
  return cached.data;
};

const getPanel = (client: Client | null, key: panel.Key): panel.Panel | undefined => {
  const cached = client?.panels.getCached(key);
  if (cached == null || cached.variant === "deleted") return undefined;
  return cached.data;
};

const subscribe = (
  { client, args: { key } }: Flux.SelectorParams<SelectKeyParams>,
  notify: () => void,
) => (client == null ? () => {} : client.panels.onChange(key, notify));

export interface SelectTabContentParams {
  key: panel.Key;
  tabKey: panel.TabKey;
}

const selectRequiredTab = (
  client: Client | null,
  { key, tabKey }: SelectTabContentParams,
): panel.Tab => {
  const tab = panel.findTab(requirePanel(client, key).root, tabKey);
  if (tab == null)
    throw new NotFoundError(`Tab with key ${tabKey} not found in panel ${key}`);
  return tab;
};

// bindTabHook lifts a hook needing both a panel key and a tab key into one whose keys are
// sourced from the surrounding Panel and Tab scopes; either may be overridden explicitly.
// The two-level analogue of scope.bindHook.
type BoundTabHook<Args extends SelectTabContentParams, R> = optional.Arg<
  optional.Optional<Args, "key" | "tabKey">,
  R
>;

const bindTabHook =
  <Args extends SelectTabContentParams, R>(
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
const bindTabSelector = <Args extends SelectTabContentParams, Selected>([
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

export interface SelectNodeParams extends SelectKeyParams {
  nodeKey: number;
}

// useSelectNodeVariant selects only the variant of the node at the given path, so a
// component that branches on split-vs-leaf does not re-render on structure changes
// within the same variant.
export const [useSelectNodeVariant, useGetNodeVariant] = Scope.bindSelector(
  Flux.createSelector<SelectNodeParams, panel.Node["variant"]>({
    subscribe,
    select: ({ client, args: { key, nodeKey } }) => {
      const node = panel.findNode(requirePanel(client, key).root, nodeKey);
      if (node == null) throw new NotFoundError(`Node at path ${nodeKey} not found`);
      return node.variant;
    },
  }),
);

// useSelectLeafNode selects the leaf node at the given path, including its tab keys.
export const [useSelectLeafNode, useGetLeafNode] = Scope.bindSelector(
  Flux.createSelector<
    SelectNodeParams,
    Omit<panel.NodeLeaf, "tabs"> & { tabs: panel.TabKey[] },
    panel.Node
  >({
    subscribe,
    select: ({ client, args: { key, nodeKey } }) => {
      const node = panel.findNode(requirePanel(client, key).root, nodeKey);
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
  Flux.createSelector<SelectNodeParams, panel.NodeSplit, panel.Node>({
    subscribe,
    select: ({ client, args: { key, nodeKey } }) => {
      const node = panel.findNode(requirePanel(client, key).root, nodeKey);
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

const tabKeys = (client: Client | null, key: panel.Key): string[] => {
  const p = getPanel(client, key);
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
// root re-renders only when tab membership changes, not on a resize or a content change.
export const [useSelectTabKeys, useGetTabKeys] = Scope.bindSelector(
  Flux.createSelector<SelectKeyParams, string[]>({
    subscribe,
    select: ({ client, args: { key } }) => tabKeys(client, key),
    equal: compare.arraysEqual,
  }),
);

// useSelectRoot selects the panel's raw stored tree root.
export const [useSelectRoot, useGetRoot] = Scope.bindSelector(
  Flux.createSelector<SelectKeyParams, panel.Node>({
    subscribe,
    select: ({ client, args: { key } }) => requirePanel(client, key).root,
  }),
);

export const [useSelectName, useGetName] = Scope.bindSelector(
  Flux.createSelector<SelectKeyParams, string>({
    subscribe,
    select: ({ client, args: { key } }) => requirePanel(client, key).name,
  }),
);

// leafTabGroups returns each leaf's ordered tab keys, or null when the panel is not
// cached.
const leafTabGroups = (
  client: Client | null,
  key: panel.Key,
): panel.TabKey[][] | null => {
  const p = getPanel(client, key);
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

export interface SelectSelectionParams {
  key?: panel.Key;
  selected: panel.TabKey[];
}

const NOOP = () => {};

const [useSelectSelectionBase, useGetSelectionBase] = Flux.createSelector<
  SelectSelectionParams,
  panel.TabKey[],
  panel.TabKey[][] | null
>({
  subscribe: ({ client, args: { key } }, notify) =>
    client == null || key == null ? NOOP : client.panels.onChange(key, notify),
  select: ({ client, args: { key } }) =>
    key == null ? null : leafTabGroups(client, key),
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
export const useSelectSelection = (params: SelectSelectionParams): panel.TabKey[] => {
  const key = Scope.useOptional(params.key);
  return useSelectSelectionBase({ ...params, key });
};

/** useGetSelection returns a getter reading {@link useSelectSelection} on demand. */
export const useGetSelection = (): ((
  params: SelectSelectionParams,
) => panel.TabKey[]) => {
  const scoped = Scope.useOptional();
  const get = useGetSelectionBase();
  return useCallback(
    (params: SelectSelectionParams) => get({ ...params, key: params.key ?? scoped }),
    [get, scoped],
  );
};

export const [useSelectTab, useGetTab] = bindTabSelector(
  Flux.createSelector<SelectTabContentParams, panel.Tab>({
    subscribe,
    select: ({ client, args }) => selectRequiredTab(client, args),
  }),
);

const selectRequiredTabLeaf = (
  client: Client | null,
  { key, tabKey }: SelectTabContentParams,
): panel.NodeLeaf => {
  const leaf = panel.findTabLeaf(requirePanel(client, key).root, tabKey);
  if (leaf == null)
    throw new NotFoundError(`Leaf holding tab ${tabKey} not found in panel ${key}`);
  return leaf;
};

// useSelectTabLeaf selects the leaf node holding the active tab. The leaf is a live
// reference into the stored tree, so immer's structural sharing gives it stable
// identity across dispatches that don't touch it.
export const [useSelectTabLeaf, useGetTabLeaf] = bindTabSelector(
  Flux.createSelector<SelectTabContentParams, panel.NodeLeaf>({
    subscribe,
    select: ({ client, args }) => selectRequiredTabLeaf(client, args),
  }),
);

// useSelectTabVariant selects only the content variant of the active tab, so a
// component that branches on resource-vs-view does not re-render on content edits
// within the same variant.
export const [useSelectTabVariant, useGetTabVariant] = bindTabSelector(
  Flux.createSelector<SelectTabContentParams, panel.TabType>({
    subscribe,
    select: ({ client, args }) => selectRequiredTab(client, args).variant,
  }),
);

// useSelectTabType selects the renderer type of the active tab: the resource's
// ontology type for resource tabs, the view type for view tabs. Components that
// render by type do not re-render when a view's args change.
export const [useSelectTabType, useGetTabType] = bindTabSelector(
  Flux.createSelector<SelectTabContentParams, string>({
    subscribe,
    select: ({ client, args }) => {
      const tab = selectRequiredTab(client, args);
      return tab.variant === "resource" ? tab.resource.type : tab.type;
    },
  }),
);

export interface SelectMaybeTabTypeParams {
  key?: panel.Key | null;
  tabKey?: panel.TabKey | null;
}

// useSelectTabResource selects the ontology ID displayed by the active resource tab.
// Resource renderers call this to learn their own document key. Throws
// UnexpectedError when the active tab is not a resource tab: only renderers mounted
// for a resource tab may call this, so a wrong-variant read is a programmer bug.
export const [useSelectTabResource, useGetTabResource] = bindTabSelector(
  Flux.createSelector<SelectTabContentParams, ontology.ID>({
    subscribe,
    select: ({ client, args }) => {
      const tab = selectRequiredTab(client, args);
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
  Flux.createSelector<SelectTabContentParams, record.Unknown>({
    subscribe,
    select: ({ client, args }) => {
      const tab = selectRequiredTab(client, args);
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

export const useList = Flux.createList<ListParams, panel.Key, panel.Panel>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.panels.retrieve(query),
  retrieveByKey: async ({ client, key }) => await client.panels.retrieve(key),
  subscribe: ({ client, query }, handler) => client.panels.onChange(query, handler),
  subscribeByKey: ({ client, key }, handler) => client.panels.onChange(key, handler),
  getCached: ({ client, query }) => client.panels.getCached(query),
});

export interface CreateParams extends panel.New {}

export const { useUpdate: useCreate } = Flux.createUpdate<CreateParams, panel.Panel>({
  name: RESOURCE_NAME,
  verbs: cache.CREATE_VERBS,
  update: async ({ client, data, onOptimisticComplete }) =>
    await client.panels.create(data, {
      onOptimistic: async ([optimistic]) => await onOptimisticComplete(optimistic),
    }),
});

export interface RenameParams extends Pick<panel.Panel, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: cache.RENAME_VERBS,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, name } = data;
    await onOptimisticComplete(data);
    await client.panels.rename(key, name);
    return data;
  },
});

export type DeleteParams = panel.Key | panel.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: cache.DELETE_VERBS,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.panels.delete(array.toArray(data), {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export const {
  useDispatch,
  useSingleDispatch: useSingleDispatchBase,
  useUndo: useUndoBase,
  useRedo: useRedoBase,
} = Flux.createDispatch<panel.Key, panel.Panel, panel.Action>({
  domain: (client) => client.panels,
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
