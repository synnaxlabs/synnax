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
  query,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { context } from "@synnaxlabs/lyra/context";
import { Haul } from "@synnaxlabs/lyra/haul";
import { useCombinedStateAndRef, useInitializerRef } from "@synnaxlabs/lyra/hooks";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree as Base } from "@synnaxlabs/lyra/tree";
import { Access, Ontology, Synnax } from "@synnaxlabs/pluto";
import { array, type destructor, type observe, unique } from "@synnaxlabs/x";
import {
  type DragEvent,
  type FC,
  type ReactElement,
  type ReactNode,
  useEffect,
  useRef,
  useState,
  useSyncExternalStore,
} from "react";
import { useStore } from "react-redux";

import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Errors } from "@/platform/errors";
import { Panel } from "@/platform/panel";
import { DefaultContextMenu } from "@/platform/tree/DefaultContextMenu";
import { DefaultItem, type Item } from "@/platform/tree/item";
import { MultipleSelectionContextMenu } from "@/platform/tree/MultipleSelectionContextMenu";
import { useItems } from "@/platform/tree/Provider";
import {
  type CreateSearchParams,
  type Match,
  type Matches,
  MatchProvider,
  type ResultNodes,
  type Search,
  type SearchAnswer,
  type SearchLayout,
  useOntologyMatches,
  type UseSearch,
} from "@/platform/tree/search";
import {
  type BaseProps,
  type ContextMenuProps,
  type GetResource,
  type TreeState,
} from "@/platform/tree/types";
import { useResources } from "@/platform/tree/useResource";
import { type Action, type State } from "@/session/store";

/** The resource types the tree sorts first and every search finds. */
const FOLDER_TYPES = new Set<ontology.ResourceType>(["group"]);

interface InternalProps {
  root: ontology.ID;
  emptyContent?: ReactNode;
  results?: ResultNodes;
  className?: string;
}

interface ContextValue {
  onDrop: (key: string, props: Haul.OnDropProps) => Haul.Item[];
  onDragStart: (itemKey: string) => void;
  onDragEnd: (e: DragEvent) => void;
  subscribeLoading: (handler: () => void) => destructor.Destructor;
  getLoading: () => string | false;
}

const [Context, useContext] = context.create<ContextValue>({
  displayName: "Tree.Context",
  providerName: "Tree.Tree",
});

const FallbackContextMenu = (): ReactElement => (
  <ContextMenu.Menu>
    <ContextMenu.ReloadConsoleItem />
  </ContextMenu.Menu>
);

interface SelectionContextMenuProps extends ContextMenuProps {
  menu: FC<ContextMenuProps>;
}

// A resource deleted while its menu is open stays in the selection. A menu reads every
// selected resource as present.
const SelectionContextMenu = ({ menu: M, ...props }: SelectionContextMenuProps) => {
  const resources = useResources(props.state, props.selection.ids);
  if (resources.length < props.selection.ids.length) return <FallbackContextMenu />;
  return <M {...props} />;
};

const TreeItem = ({ onDrop: _, ...rest }: Base.ItemProps<string>) => {
  const { itemKey } = rest;
  const id = ontology.parseID(itemKey);
  const resource = List.useItem<string, ontology.Resource>(itemKey);
  const Item = useItems()[id.type] ?? DefaultItem;
  const { onDrop, subscribeLoading, getLoading, onDragStart, onDragEnd } =
    useContext("Tree.itemRenderProp");
  const handleDragStart = () => onDragStart(itemKey);
  const loading = useSyncExternalStore(
    subscribeLoading,
    () => getLoading() === itemKey,
  );
  // A drop re-parents the dragged items, so the grant is read on them and not on the
  // destination. The item itself carries the same grant to leave its parent.
  const client = Synnax.use();
  const canMove = Access.useUpdateGranted(id);

  const [draggingOver, setDraggingOver] = useState(false);

  const onDropDrops = Haul.useDrop({
    type: Base.HAUL_TYPE,
    key: itemKey,
    canDrop: ({ items: entities, source }) => {
      const keys = entities.map((item) => item.key);
      setDraggingOver(false);
      if (source.type !== Base.HAUL_TYPE || keys.includes(itemKey)) return false;
      return Access.updateGranted({
        client,
        id: Base.filterHaulItems(entities).map(({ key }) => ontology.idZ.parse(key)),
      });
    },
    onDrop: (props) => onDrop(itemKey, props) ?? [],
    onDragOver: () => setDraggingOver(true),
  });

  // The virtualizer has already reserved this row's slot. Rendering nothing would
  // leave a gap, so the row stays empty until the resource loads.
  if (resource == null) return <Base.Item {...rest} />;

  return (
    <Item
      {...rest}
      draggingOver={draggingOver}
      onDragStart={handleDragStart}
      draggable={canMove}
      {...onDropDrops}
      onDragLeave={() => setDraggingOver(false)}
      onDragEnd={onDragEnd}
      resource={resource}
      loading={loading}
    />
  );
};

const itemRenderProp = Component.renderProp(TreeItem);

// Returns the keys that sit inside another key in the same set. Depth cannot answer
// this: two nodes at different depths are often unrelated.
const findContainedKeys = (tree: Base.Node<string>[], keys: string[]): Set<string> => {
  const all = new Set(keys);
  const contained = new Set<string>();
  keys.forEach((key) => {
    const node = Base.findNode({ tree, key });
    if (node?.children == null) return;
    Base.getDescendants(...node.children).forEach(({ key: descendant }) => {
      if (all.has(descendant)) contained.add(descendant);
    });
  });
  return contained;
};

// A parent's child list says nothing about each child's own subtree, so a node
// that is already in the tree keeps the children it has already loaded.
const keepLoadedChildren = (
  prev: Base.Node<string>[],
  next: Base.Node<string>[],
): Base.Node<string>[] =>
  next.map((node) => {
    const existing = prev.find(({ key }) => key === node.key);
    return existing == null ? node : { ...node, children: existing.children };
  });

// A watched group keeps the children its answer gave, since they are every child and
// not only the matches.
const keepWatchedChildren = (
  prev: Base.Node<string>[],
  next: Base.Node<string>[],
  watched: Map<string, destructor.Destructor>,
): Base.Node<string>[] =>
  next.map((node) => {
    const existing = prev.find(({ key }) => key === node.key);
    if (existing?.children == null || node.children == null) return node;
    if (watched.has(node.key)) return { ...node, children: existing.children };
    return {
      ...node,
      children: keepWatchedChildren(existing.children, node.children, watched),
    };
  });

const toNodes = (
  resources: ontology.Resource[],
  resolveItem: (type: ontology.ResourceType) => Item,
): Base.Node<string>[] =>
  resources
    .filter((r) => {
      const svc = resolveItem(r.id.type);
      return svc.visible == null || svc.visible(r);
    })
    .map((r) => ({
      key: ontology.idToString(r.id),
      children: resolveItem(r.id.type).hasChildren ? [] : undefined,
    }));

const Internal = ({
  root,
  emptyContent,
  results,
  className,
}: InternalProps): ReactElement => {
  const items = useItems();
  const resolveItem = (type: ontology.ResourceType): Item => items[type] ?? DefaultItem;
  const [selected, setSelected, selectedRef] = useCombinedStateAndRef<string[]>([]);
  const loadingRef = useRef<string | false>(false);
  const handleError = Status.useErrorHandler();
  const client = Synnax.use();
  // The retained answer seeds the first render, not an effect: an effect runs
  // after the commit paints, so the tree would still show one empty frame.
  const [seed] = useState<Base.Node<string>[] | null>(() => {
    if (results != null) return null;
    const cached = client?.ontology.children.getCached({ ids: root });
    return query.isLive(cached) ? toNodes(cached, resolveItem) : null;
  });
  const [nodes, setNodes, nodesRef] = useCombinedStateAndRef<Base.Node<string>[]>(
    seed ?? [],
  );
  // An unanswered root has no children yet and no empty content: they are the
  // same node list, and only the answer tells them apart.
  const [answered, setAnswered] = useState(seed != null);
  const loadingListenersRef = useInitializerRef(() => new Set<observe.Handler<void>>());

  // Placeholder resources back tree items (e.g. a just-created group awaiting its
  // inline rename) before the Core delivers the real resource.
  const placeholders = List.useMapData<string, ontology.Resource>();

  const getResourceByKey = (key: string): ontology.Resource | undefined =>
    client?.ontology.cache.resources.get(key) ?? placeholders.getItem(key);

  const subscribeResource = (
    handler: () => void,
    key: string,
  ): destructor.Destructor => {
    const releasePlaceholder = placeholders.subscribe(handler, key);
    const releaseCached = client?.ontology.cache.resources.subscribe(handler, key);
    return () => {
      releasePlaceholder();
      releaseCached?.();
    };
  };

  const sort = (a: Base.Node<string>, b: Base.Node<string>) => {
    const aResource = getResourceByKey(a.key);
    const bResource = getResourceByKey(b.key);
    if (aResource == null && bResource == null) return 0;
    if (aResource == null) return 1;
    if (bResource == null) return -1;
    const aFolder = FOLDER_TYPES.has(aResource.id.type);
    const bFolder = FOLDER_TYPES.has(bResource.id.type);
    if (aFolder !== bFolder) return aFolder ? -1 : 1;
    return aResource.name.localeCompare(bResource.name);
  };

  const applyAnswer = (parent: ontology.ID, resources: ontology.Resource[]) => {
    const next = toNodes(resources, resolveItem);
    const nextKeys = new Set(next.map(({ key }) => key));
    // A placeholder stands in only until the Core delivers the real resource, which
    // the answer carries.
    const settled = ontology
      .idToString(resources.map(({ id }) => id))
      .filter(placeholders.hasItem);
    if (settled.length > 0) placeholders.deleteItem(settled);
    // The answer is the authority on its parent's membership. A node it omits
    // survives only while a placeholder backs it, since an optimistic row the Core
    // has not heard about yet cannot be in any answer.
    const merge = (prevChildren: Base.Node<string>[]): Base.Node<string>[] => {
      const merged = [
        ...prevChildren.filter(
          ({ key }) => !nextKeys.has(key) && placeholders.hasItem(key),
        ),
        ...keepLoadedChildren(prevChildren, next),
      ];
      // Results keep the order of the matches, so the tree does not sort them, and
      // the children of a group arrive sorted instead.
      return results == null ? merged : merged.sort(sort);
    };
    if (ontology.idsEqual(parent, root)) {
      setNodes(merge);
      setAnswered(true);
      return;
    }
    setNodes((prevNodes) => [
      // The children subscription can outlive the parent's presence in the
      // tree; a missing parent means the update is stale.
      ...Base.updateNodeChildren({
        tree: prevNodes,
        parent: ontology.idToString(parent),
        throwOnMissing: false,
        updater: merge,
      }),
    ]);
  };

  const subscribeLoading = (handler: () => void) => {
    loadingListenersRef.current.add(handler);
    return () => {
      loadingListenersRef.current.delete(handler);
    };
  };

  const getLoading = () => loadingRef.current;

  const setLoading = (key: string | false) => {
    loadingRef.current = key;
    loadingListenersRef.current.forEach((callback) => callback());
  };

  /** Live children subscriptions, keyed by the parent they answer for. */
  const watchedRef = useInitializerRef(() => new Map<string, destructor.Destructor>());

  const watchChildren = async (parent: ontology.ID): Promise<void> => {
    if (client == null) return;
    const key = ontology.idToString(parent);
    // An already-watched parent has nothing to wait for, and no fetch will come
    // along to end the wait.
    if (watchedRef.current.has(key)) {
      if (loadingRef.current === key) setLoading(false);
      return;
    }
    // Subscribe before fetching. A change landing between the two would
    // otherwise reach neither the answer nor the tree.
    watchedRef.current.set(
      key,
      client.ontology.children.onChange({ ids: parent }, (answer) => {
        if (query.isLive(answer)) applyAnswer(parent, answer);
      }),
    );
    // A retained answer paints before the fetch that reconfirms it.
    const cached = client.ontology.children.getCached({ ids: parent });
    if (query.isLive(cached)) applyAnswer(parent, cached);
    try {
      const resources = await client.ontology.children.retrieve({ ids: parent });
      // A release during the fetch means the answer is no longer wanted.
      if (watchedRef.current.has(key)) applyAnswer(parent, resources);
    } catch (e) {
      handleError(e, "Failed to retrieve resources");
    }
    if (loadingRef.current === key) setLoading(false);
  };

  const releaseChildren = (keys: string[]) => {
    keys.forEach((key) => {
      watchedRef.current.get(key)?.();
      watchedRef.current.delete(key);
    });
  };

  useEffect(() => {
    if (results == null) void watchChildren(root);
    return () => releaseChildren([...watchedRef.current.keys()]);
  }, [client, ontology.idToString(root)]);

  // Selection drives the context menu, and a key with no node poisons every later
  // right-click, so a node that leaves the tree leaves the selection with it.
  useEffect(() => {
    setSelected((prev) => {
      const next = prev.filter((key) => Base.findNode({ tree: nodes, key }) != null);
      return next.length === prev.length ? prev : next;
    });
  }, [nodes, setSelected]);

  const handleExpand = ({ action, clicked }: Base.HandleExpandProps) => {
    if (action === "contract") {
      // A hidden subtree needs no maintenance. Its answers stay cached, so a
      // re-expand paints from them.
      const node = Base.findNode({ tree: nodesRef.current, key: clicked });
      if (node != null)
        releaseChildren(Base.getDescendants(node).map(({ key }) => key));
      return;
    }
    if (results?.fixed.has(clicked) === true) return;
    setLoading(clicked);
    void watchChildren(ontology.parseID(clicked));
  };

  const getResource = ((id: ontology.ID | ontology.ID[] | string | string[]) => {
    const isSingle = !Array.isArray(id);
    const ids = array.toArray(id);
    const resources = ontology
      .idToString(ids)
      .map(getResourceByKey)
      .filter((r) => r != null);
    if (isSingle) {
      if (resources[0] == null)
        throw new NotFoundError(`Resource ${ontology.idToString(id)} not found`);
      return resources[0];
    }

    return resources;
  }) as GetResource;

  const getItem = List.createGetItem<string, ontology.Resource>(
    getResourceByKey,
    (keys) => keys.map(getResourceByKey).filter((r) => r != null),
  );

  const treeProps = Base.use({
    nodes,
    onExpand: handleExpand,
    selected,
    onSelectedChange: setSelected,
    sort: results == null ? sort : undefined,
  });
  const { shape, expand, contract } = treeProps;

  // Groups open on arrival once, so a group the user closes stays closed.
  const openedRef = useInitializerRef(() => new Set<string>());
  useEffect(() => {
    if (results == null) return;
    // A group that now holds only matches drops every child its watch gave.
    releaseChildren(
      [...watchedRef.current.keys()].filter((key) => results.fixed.has(key)),
    );
    setNodes((prev) => keepWatchedChildren(prev, results.nodes, watchedRef.current));
    results.expanded.forEach((key) => {
      if (openedRef.current.has(key)) return;
      openedRef.current.add(key);
      expand(key);
    });
  }, [results]);
  // A placeholder's row stays mounted, so its rename can start out of view. Focusing
  // the editable then scrolls the row into view.
  const pinned = shape.keys.filter(placeholders.hasItem);

  const state: TreeState = {
    nodes,
    shape,
    setNodes,
    expand,
    contract,
    setResource: placeholders.setItem,
    getResource,
    subscribeResource,
    setSelection: setSelected,
  };

  const openTab = Panel.useOpenTab();
  const addStatus = Status.useAdder();
  const store = useStore<State, Action>();

  const { update: moveChildren } = Ontology.useMoveChildren({});

  // A result at the top of the tree has a real parent the tree does not show.
  const parentOf = (tree: Base.Node<string>[], key: string): ontology.ID => {
    const parent = Base.findNodeParent({ tree, key });
    if (parent != null) return ontology.parseID(parent.key);
    return results?.parents.get(key) ?? root;
  };

  const getBaseProps = (client: Client): BaseProps => ({
    client,
    store,
    openTab,
    addStatus,
    handleError,
  });

  const handleDrop = (
    key: string,
    { source, items }: Haul.OnDropProps,
  ): Haul.Item[] => {
    const nodesSnapshot = nodesRef.current;
    const dropped = Base.filterHaulItems(items);
    const isValidDrop = dropped.length > 0 && source.type === Base.HAUL_TYPE;
    if (!isValidDrop) return [];
    const destination = ontology.parseID(key);
    const svc = resolveItem(destination.type);
    if (!svc.canDrop({ source, items })) return [];

    // An item inside another dragged item travels with it. Moving it explicitly
    // would re-parent it onto the destination as a sibling of its own parent.
    const droppedKeys = dropped.map(({ key }) => key);
    const contained = findContainedKeys(nodesSnapshot, droppedKeys);
    const moved = dropped.filter(({ key }) => !contained.has(key));
    if (moved.length === 0) return [];

    // Each item leaves its own parent: a selection can span several.
    const bySource = new Map<string, ontology.ID[]>();
    moved.forEach(({ key }) => {
      const sourceKey = ontology.idToString(parentOf(nodesSnapshot, key));
      const ids = bySource.get(sourceKey) ?? [];
      ids.push(ontology.parseID(key));
      bySource.set(sourceKey, ids);
    });
    contract(...moved.map(({ key }) => key));
    bySource.forEach((ids, sourceKey) =>
      moveChildren({
        source: ontology.parseID(sourceKey),
        destination,
        ids,
      }),
    );
    return moved;
  };

  const { startDrag, onDragEnd } = Haul.useDrag({ type: Base.HAUL_TYPE });

  const handleDragStart = (itemKey: string) => {
    if (selectedRef.current.includes(itemKey)) {
      const selectedHaulItems = getResource(selectedRef.current).flatMap((res) => {
        const items: Haul.Item[] = [Base.createHaulItem(ontology.idToString(res.id))];
        const svcItems = resolveItem(res.id.type).haulItems(res);
        if (svcItems != null) items.push(...svcItems);
        return items;
      });
      return startDrag(selectedHaulItems);
    }
    const haulItems = resolveItem(ontology.parseID(itemKey).type).haulItems(
      getResource(itemKey),
    );
    startDrag([Base.createHaulItem(itemKey), ...haulItems]);
  };

  const renderContextMenu = ({ keys }: Menu.ContextMenuMenuProps) => {
    if (client == null) return <FallbackContextMenu />;
    if (keys.length === 0) return <DefaultContextMenu root={root} state={state} />;
    const rightClickedButNotSelected = keys.find((v) => !selected.includes(v));
    // In the case where we right clicked the menu, but it's not in the current
    // selection, we only display a context menu for that item.
    if (rightClickedButNotSelected != null) keys = [rightClickedButNotSelected];
    // Because we're using a virtualized tree, the keys from the context menu
    // might not actually be accurate (because we're missing DOM elements), so instead
    // we pull directly from the list selected state.
    else keys = selected;

    const ids = keys.map((key) => ontology.parseID(key));

    // We want to find the parent of the node with the lowest depth, since we
    // might be selecting nodes AND their children.
    const [shallowest] = keys.toSorted(Base.compareDepth(shape));
    // TODO: we might be selecting two nodes that are not ascendants or descendants of
    // the other ones. We need to change this function to implement recursion.
    const parentID = parentOf(nodes, shallowest);

    const firstID = ontology.parseID(shallowest);

    const props: ContextMenuProps = {
      selection: {
        rootID: root,
        parentID,
        ids,
      },
      state,
      ...getBaseProps(client),
    };

    const allSameType = ids.every((id) => id.type === firstID.type);
    const M = allSameType
      ? resolveItem(firstID.type).ContextMenu
      : MultipleSelectionContextMenu;
    return M == null ? (
      <FallbackContextMenu />
    ) : (
      <SelectionContextMenu menu={M} {...props} />
    );
  };
  const handleContextMenu = (props: Menu.ContextMenuMenuProps) => (
    <Errors.SuspenseBoundary loading={null}>
      {renderContextMenu(props)}
    </Errors.SuspenseBoundary>
  );
  const menuProps = Menu.useContextMenu();
  const contextValue = {
    onDrop: handleDrop,
    subscribeLoading,
    getLoading,
    onDragStart: handleDragStart,
    onDragEnd,
  };

  return (
    <Context value={contextValue}>
      <Menu.ContextMenu menu={handleContextMenu} {...menuProps} />
      <Base.Tree<string, ontology.Resource>
        {...treeProps}
        showRules
        shape={shape}
        subscribe={subscribeResource}
        // Not getResource: it throws, and a resource may not be cached before
        // the tree attempts to render it.
        getItem={getItem}
        emptyContent={answered || results != null ? emptyContent : null}
        className={className}
        pinned={pinned}
        onContextMenu={menuProps.open}
      >
        {itemRenderProp}
      </Base.Tree>
    </Context>
  );
};

const LIMIT = 100;
const DEBOUNCE_MS = 120;
const EMPTY_ANCESTRY: ontology.Ancestry = { resources: [], paths: {} };

type FoundMatches = Extract<Matches, { variant: "success" }>;
type FailedMatches = Extract<Matches, { variant: "error" }>;

const isFound = (matches: Matches): matches is FoundMatches =>
  matches.variant === "success";

const isFailed = (matches: Matches): matches is FailedMatches =>
  matches.variant === "error";

const isFolderNode = ({ key }: Base.Node<string>): boolean =>
  FOLDER_TYPES.has(ontology.parseID(key).type);

const foldersFirst = (nodes: Base.Node<string>[]): Base.Node<string>[] =>
  [...nodes.filter(isFolderNode), ...nodes.filter((node) => !isFolderNode(node))].map(
    (node) =>
      node.children == null ? node : { ...node, children: foldersFirst(node.children) },
  );

interface BuildResultsParams {
  ids: ontology.ID[];
  ancestry: ontology.Ancestry;
  layout: SearchLayout;
  root: ontology.ID;
  hasChildren: (id: ontology.ID) => boolean;
}

interface Built {
  results: ResultNodes;
  groups: Map<string, string>;
}

const buildResults = ({
  ids,
  ancestry: { resources, paths },
  layout,
  root,
  hasChildren,
}: BuildResultsParams): Built => {
  const names = new Map(resources.map(({ key, name }) => [key, name]));
  const matched = new Set(ontology.idToString(ids));
  const nodes: Base.Node<string>[] = [];
  const parents = new Map<string, ontology.ID>();
  const groups = new Map<string, string>();
  const fixed = new Set<string>();
  const expanded = new Set<string>();
  const toNode = (id: ontology.ID): Base.Node<string> => ({
    key: ontology.idToString(id),
    children: hasChildren(id) ? [] : undefined,
  });
  ids.forEach((id) => {
    const key = ontology.idToString(id);
    const idPaths = paths[key] ?? [];
    // The tree shows only what sits under its root.
    if (idPaths.length === 0) return;
    const parent = idPaths[0].at(-1);
    parents.set(key, parent ?? root);
    if (layout === "flat") {
      const group = parent == null ? undefined : names.get(ontology.idToString(parent));
      if (group != null) groups.set(key, group);
      nodes.push(toNode(id));
      return;
    }
    idPaths.forEach((path) => {
      let level = nodes;
      path.forEach((groupID) => {
        const groupKey = ontology.idToString(groupID);
        expanded.add(groupKey);
        if (!matched.has(groupKey)) fixed.add(groupKey);
        let node = level.find((n) => n.key === groupKey);
        if (node == null) {
          node = { key: groupKey, children: [] };
          level.push(node);
        }
        node.children ??= [];
        level = node.children;
      });
      if (!level.some((n) => n.key === key)) level.push(toNode(id));
    });
  });
  return {
    results: { nodes: foldersFirst(nodes), fixed, expanded: [...expanded], parents },
    groups,
  };
};

interface Held {
  key: string;
  layout: SearchLayout;
  ancestry: ontology.Ancestry;
  results: ResultNodes;
  match: Match;
  more: boolean;
}

const useDebounced = (value: string): string => {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const timeout = setTimeout(() => setDebounced(value), DEBOUNCE_MS);
    return () => clearTimeout(timeout);
  }, [value]);
  return debounced;
};

/**
 * Declares the search of a tree. Call it at module scope, and call the hook it returns
 * where both the search field and the tree can read its state.
 */
export const createSearch = ({
  noun,
  types = [],
  useMatches = useOntologyMatches,
}: CreateSearchParams): UseSearch => {
  const searchTypes = [...FOLDER_TYPES, ...types];
  const useSearch = (root: ontology.ID | null): Search => {
    const items = useItems();
    const [open, setOpen] = useState(false);
    const [term, setTerm] = useState("");
    const [layout, setLayout] = useState<SearchLayout>("flat");
    const searching = open && term.trim() !== "";
    const trimmed = useDebounced(term).trim();
    const sources = useMatches(
      !open || root == null || trimmed === ""
        ? null
        : { term: trimmed, root, types: searchTypes, limit: LIMIT + 1 },
    );
    const found = sources.every(isFound)
      ? unique.by(
          sources.flatMap(({ ids }) => ids),
          (id) => ontology.idToString(id),
        )
      : null;
    const shownIDs = found?.slice(0, LIMIT) ?? null;
    const ancestry = Ontology.useResultAncestors(
      root == null || shownIDs == null || shownIDs.length === 0
        ? null
        : { ids: shownIDs, root },
    );
    let placed: ontology.Ancestry | null = null;
    if (shownIDs?.length === 0) placed = EMPTY_ANCESTRY;
    else if (ancestry.variant === "success") placed = ancestry.data;
    const key =
      shownIDs == null ? null : [trimmed, ...ontology.idToString(shownIDs)].join("\n");
    const more = found != null && found.length > LIMIT;

    // The last answer shows until the next one is placed, so the results never blank
    // between keystrokes.
    const [held, setHeld] = useState<Held | null>(null);
    if (!searching) {
      if (held != null) setHeld(null);
    } else if (
      root != null &&
      key != null &&
      found != null &&
      shownIDs != null &&
      placed != null &&
      (held?.key !== key ||
        held.ancestry !== placed ||
        held.layout !== layout ||
        held.more !== more)
    ) {
      const { results, groups } = buildResults({
        ids: shownIDs,
        ancestry: placed,
        layout,
        root,
        hasChildren: (id) => (items[id.type] ?? DefaultItem).hasChildren,
      });
      setHeld({
        key,
        layout,
        ancestry: placed,
        results,
        match: { term: trimmed, groups },
        more,
      });
    }

    const failure =
      sources.find(isFailed) ?? (ancestry.variant === "error" ? ancestry.status : null);
    let answer: SearchAnswer | null = null;
    if (searching && failure != null)
      answer = { variant: "error", description: failure.description };
    else if (searching && held != null)
      answer = {
        variant: "success",
        results: held.results,
        match: held.match,
        more: held.more,
      };
    return {
      noun,
      open,
      term,
      layout,
      loading: searching && failure == null && (key == null || held?.key !== key),
      answer,
      start: () => setOpen(true),
      close: () => {
        setOpen(false);
        setTerm("");
      },
      setTerm,
      setLayout,
    };
  };
  return useSearch;
};

interface NoMatchesProps {
  noun: string;
  term: string;
}

const NoMatches = ({ noun, term }: NoMatchesProps): ReactElement => (
  <Text.Text level="small" color={9} className={CSS.BE("tree-search", "message")}>
    No {noun} match “{term}”
  </Text.Text>
);

interface ResultsProps {
  root: ontology.ID;
  noun: string;
  layout: SearchLayout;
  answer: SearchAnswer;
}

const Results = ({ root, noun, layout, answer }: ResultsProps): ReactElement => {
  if (answer.variant === "error")
    return (
      <Status.Summary
        className={CSS.BE("tree-search", "message")}
        variant="error"
        message={`Failed to search ${noun}`}
        description={answer.description}
      />
    );
  return (
    <MatchProvider value={answer.match}>
      <Internal
        key={layout}
        root={root}
        results={answer.results}
        emptyContent={<NoMatches noun={noun} term={answer.match.term} />}
      />
      {answer.more && (
        <Text.Text level="small" color={9} className={CSS.BE("tree-search", "more")}>
          Showing the first {LIMIT} matches. Type more to narrow the search.
        </Text.Text>
      )}
    </MatchProvider>
  );
};

export interface TreeProps {
  root?: ontology.ID | null;
  emptyContent?: ReactNode;
  /** Shows the search answer in place of the tree while the search has one. */
  search?: Search;
}

export const Tree = ({
  root,
  emptyContent,
  search,
}: TreeProps): ReactElement | null => {
  if (root == null) return null;
  return (
    <>
      {/* The results come ahead of the hidden tree, so a lookup by item ID, such as
          the start of a rename, finds the result first. */}
      {search?.answer != null && (
        <Results
          root={root}
          noun={search.noun}
          layout={search.layout}
          answer={search.answer}
        />
      )}
      <Internal
        root={root}
        emptyContent={emptyContent}
        className={search?.answer != null ? CSS.BM("tree", "hidden") : undefined}
      />
    </>
  );
};
