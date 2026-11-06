// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type group,
  NotFoundError,
  type ontology,
  resource,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { context } from "@synnaxlabs/lyra/context";
import { Haul } from "@synnaxlabs/lyra/haul";
import {
  useCombinedStateAndRef,
  useInitializerRef,
  useSyncedRef,
} from "@synnaxlabs/lyra/hooks";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import { Store } from "@synnaxlabs/lyra/store";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree as Base } from "@synnaxlabs/lyra/tree";
import { Access, Group, Synnax } from "@synnaxlabs/pluto";
import { array, compare, type destructor, type observe, unique } from "@synnaxlabs/x";
import {
  type DragEvent,
  type FC,
  type KeyboardEvent,
  type ReactElement,
  type ReactNode,
  useEffect,
  useRef,
  useState,
  useSyncExternalStore,
} from "react";
import { flushSync } from "react-dom";
import { useStore } from "react-redux";

import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Errors } from "@/platform/errors";
import { Panel } from "@/platform/panel";
import { type Source } from "@/platform/tree/children";
import {
  DefaultContextMenu,
  type DefaultContextMenuProps,
} from "@/platform/tree/DefaultContextMenu";
import { DefaultItem, type Item } from "@/platform/tree/item";
import { MultipleSelectionContextMenu } from "@/platform/tree/MultipleSelectionContextMenu";
import { useItems } from "@/platform/tree/Provider";
import {
  type CreateSearchParams,
  groupsAbove,
  type Match,
  type Matches,
  matchKey,
  MatchProvider,
  placeInGroups,
  type Placement,
  type ResultNodes,
  type Search,
  type SearchAnswer,
  type SearchLayout,
  useGroupMatches,
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

/** The resource types the tree sorts first. */
const FOLDER_TYPES = new Set<resource.Type>(["group", "arc_directory"]);

interface InternalProps extends Pick<List.ScrollProps, "ref" | "onKeyDown"> {
  root: resource.ID;
  /**
   * Lists the children of the root. Defaults to the children the item of the root's
   * type lists, for a root that is a resource.
   */
  listRoot?: (client: Client) => Source[];
  emptyContent?: ReactNode;
  /** The menu for a right-click on empty space. */
  rootContextMenu?: FC<DefaultContextMenuProps>;
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
  const id = resource.parseID(itemKey);
  const ontologyResource = List.useItem<string, ontology.Resource>(itemKey);
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
        id: Base.filterHaulItems(entities).map(({ key }) => resource.idZ.parse(key)),
      });
    },
    onDrop: (props) => onDrop(itemKey, props) ?? [],
    onDragOver: () => setDraggingOver(true),
  });

  // The virtualizer has already reserved this row's slot. Rendering nothing would
  // leave a gap, so the row stays empty until the resource loads.
  if (ontologyResource == null) return <Base.Item {...rest} />;

  return (
    <Item
      {...rest}
      draggingOver={draggingOver}
      onDragStart={handleDragStart}
      draggable={canMove}
      {...onDropDrops}
      onDragLeave={() => setDraggingOver(false)}
      onDragEnd={onDragEnd}
      resource={ontologyResource}
      loading={loading}
    />
  );
};

const itemRenderProp = Component.renderProp(TreeItem);

// Returns the keys that sit inside another key in the same set, in the tree or in the
// ancestors of a search match. Depth cannot answer this: two nodes at different depths
// are often unrelated.
const findContainedKeys = (
  tree: Base.Node<string>[],
  keys: string[],
  ancestors: Map<string, Set<string>> = new Map(),
): Set<string> => {
  const all = new Set(keys);
  const contained = new Set<string>();
  keys.forEach((key) => {
    if ([...(ancestors.get(key) ?? [])].some((a) => all.has(a))) contained.add(key);
    const node = Base.findNode({ tree, key });
    if (node?.children == null) return;
    Base.getDescendants(...node.children).forEach(({ key: descendant }) => {
      if (all.has(descendant)) contained.add(descendant);
    });
  });
  return contained;
};

// A list of children says nothing about each child's own subtree, so a node already in
// the tree, under this parent or another, keeps the children it has loaded.
const keepLoadedChildren = (
  tree: Base.Node<string>[],
  next: Base.Node<string>[],
): Base.Node<string>[] => {
  const loaded = new Map<string, Base.Node<string>[]>();
  const index = (nodes: Base.Node<string>[]) =>
    nodes.forEach(({ key, children }) => {
      if (children == null) return;
      loaded.set(key, children);
      index(children);
    });
  index(tree);
  return next.map((node) => {
    const children = loaded.get(node.key);
    return children == null ? node : { ...node, children };
  });
};

// Updates the children of every node under the key, since one resource can sit under
// several parents.
const updateChildren = (
  tree: Base.Node<string>[],
  key: string,
  updater: (children: Base.Node<string>[]) => Base.Node<string>[],
): Base.Node<string>[] => {
  let changed = false;
  const next = tree.map((node) => {
    if (node.key === key) {
      changed = true;
      return { ...node, children: updater(node.children ?? []) };
    }
    if (node.children == null) return node;
    const children = updateChildren(node.children, key, updater);
    if (children === node.children) return node;
    changed = true;
    return { ...node, children };
  });
  return changed ? next : tree;
};

/** The live list of the children of a node. */
interface Watch {
  sources: Source[];
  /** The latest answer of each source, undefined until the source answers. */
  answers: Array<ontology.Resource[] | undefined>;
  /** The keys of the latest whole list. */
  keys: Set<string>;
  /** The keys of the visible children of the latest whole list, in sorted order. */
  order?: string[];
  /** Ends the subscriptions of the sources. Absent while the node is collapsed. */
  release?: destructor.Destructor;
}

interface Answer {
  resources: ontology.Resource[];
  /** The source that lists each key. */
  sources: Map<string, Source>;
}

// A list of children is whole only once every source has answered.
const combine = ({ sources, answers }: Watch): Answer | null => {
  const answer: Answer = { resources: [], sources: new Map() };
  for (const [i, resources] of answers.entries()) {
    if (resources == null) return null;
    resources.forEach((r) => {
      answer.resources.push(r);
      answer.sources.set(resource.idToString(r.id), sources[i]);
    });
  }
  return answer;
};

const createWatch = (sources: Source[]): Watch => {
  const watch: Watch = {
    sources,
    answers: sources.map((source) => source.getCached()),
    keys: new Set(),
  };
  watch.keys = new Set(combine(watch)?.sources.keys());
  return watch;
};

interface Owners {
  /** Returns the source that rows read the key from. */
  ownerOf: (key: string) => Source | undefined;
  has: (key: string) => boolean;
  /** Each returns whether the source that rows read the key from changed. */
  own: (key: string, parent: string, source: Source) => boolean;
  disown: (key: string, parent: string) => boolean;
  /** Tells the rows of the keys that their source changed. */
  notify: (keys: string[]) => void;
  subscribe: (handler: () => void, key: string) => destructor.Destructor;
}

// Holds the sources that list each key, by the key of the parent that lists it.
const useOwners = (seed: Watch | null, rootKey: string): Owners => {
  const ownersRef = useInitializerRef(() => {
    const owners = new Map<string, Map<string, Source>>();
    if (seed == null) return owners;
    combine(seed)?.sources.forEach((source, key) =>
      owners.set(key, new Map([[rootKey, source]])),
    );
    return owners;
  });
  const { notifyListeners, subscribe } = Store.useKeyedListeners<string>();
  const ownerOf = (key: string): Source | undefined =>
    ownersRef.current.get(key)?.values().next().value;
  const own = (key: string, parent: string, source: Source): boolean => {
    const before = ownerOf(key);
    const owners = ownersRef.current.get(key) ?? new Map<string, Source>();
    owners.set(parent, source);
    ownersRef.current.set(key, owners);
    return ownerOf(key) !== before;
  };
  const disown = (key: string, parent: string): boolean => {
    const owners = ownersRef.current.get(key);
    if (owners?.delete(parent) !== true) return false;
    if (owners.size === 0) ownersRef.current.delete(key);
    return true;
  };
  return {
    ownerOf,
    has: (key) => ownersRef.current.has(key),
    own,
    disown,
    notify: notifyListeners,
    subscribe,
  };
};

// Reads the resource of every node once, since a sort compares each one many times.
const readResources = (
  nodes: Base.Node<string>[],
  read: (key: string) => ontology.Resource | undefined,
): Map<string, ontology.Resource> => {
  const resources = new Map<string, ontology.Resource>();
  const visit = (level: Base.Node<string>[]) =>
    level.forEach(({ key, children }) => {
      const res = read(key);
      if (res != null) resources.set(key, res);
      if (children != null) visit(children);
    });
  visit(nodes);
  return resources;
};

const filterVisible = (
  resources: ontology.Resource[],
  resolveItem: (type: resource.Type) => Item,
): ontology.Resource[] =>
  resources.filter((r) => {
    const { visible } = resolveItem(r.id.type);
    return visible == null || visible(r);
  });

// No answer holds a node the Core has not heard of yet, so a node that a placeholder
// backs stays. The nodes moved under it leave the answer.
const keepPlaceholders = (
  prev: Base.Node<string>[],
  next: Base.Node<string>[],
  isPlaceholder: (key: string) => boolean,
): Base.Node<string>[] => {
  const nextKeys = new Set(next.map(({ key }) => key));
  const kept = prev.filter(({ key }) => !nextKeys.has(key) && isPlaceholder(key));
  const moved = new Set(Base.getDescendants(...kept).map(({ key }) => key));
  return [...kept, ...next.filter(({ key }) => !moved.has(key))];
};

// A watched group keeps the children its answer gave, since they are every child and
// not only the matches.
const keepWatchedChildren = (
  prev: Base.Node<string>[],
  next: Base.Node<string>[],
  watched: Map<string, unknown>,
  isPlaceholder: (key: string) => boolean,
): Base.Node<string>[] =>
  keepPlaceholders(
    prev,
    next.map((node) => {
      const existing = prev.find(({ key }) => key === node.key);
      if (existing?.children == null || node.children == null) return node;
      if (watched.has(node.key)) return { ...node, children: existing.children };
      return {
        ...node,
        children: keepWatchedChildren(
          existing.children,
          node.children,
          watched,
          isPlaceholder,
        ),
      };
    }),
    isPlaceholder,
  );

const toNodes = (
  resources: ontology.Resource[],
  resolveItem: (type: resource.Type) => Item,
): Base.Node<string>[] =>
  filterVisible(resources, resolveItem).map((r) => ({
    key: resource.idToString(r.id),
    children: resolveItem(r.id.type).hasChildren ? [] : undefined,
  }));

// Folders come first, then resources by name.
const compareResources = (a: ontology.Resource, b: ontology.Resource): number => {
  const aFolder = FOLDER_TYPES.has(a.id.type);
  const bFolder = FOLDER_TYPES.has(b.id.type);
  if (aFolder !== bFolder) return aFolder ? -1 : 1;
  return a.name.localeCompare(b.name);
};

const Internal = ({
  root,
  listRoot,
  emptyContent,
  rootContextMenu: RootContextMenu = DefaultContextMenu,
  results,
  className,
  ref,
  onKeyDown,
}: InternalProps): ReactElement => {
  const items = useItems();
  const resolveItem = (type: resource.Type): Item => items[type] ?? DefaultItem;
  const [selected, setSelected, selectedRef] = useCombinedStateAndRef<string[]>([]);
  const loadingRef = useRef<string | false>(false);
  const handleError = Status.useErrorHandler();
  const client = Synnax.use();
  const rootKey = resource.idToString(root);
  // Placeholder resources back tree items (e.g. a just-created group awaiting its
  // inline rename) before the Core delivers the real resource.
  const placeholders = List.useMapData<string, ontology.Resource>();

  const listChildren = (client: Client, id: resource.ID): Source[] => {
    if (listRoot != null && resource.idsEqual(id, root)) return listRoot(client);
    const { children } = resolveItem(id.type);
    if (children == null) throw new Error(`no children resolver for ${id.type}`);
    return children({ client, id, root });
  };

  // The retained answer seeds the first render, not an effect: an effect runs
  // after the commit paints, so the tree would still show one empty frame.
  const [seed] = useState<Watch | null>(() =>
    client == null || results != null ? null : createWatch(listChildren(client, root)),
  );
  const watchesRef = useInitializerRef(
    () => new Map<string, Watch>(seed == null ? [] : [[rootKey, seed]]),
  );
  const owners = useOwners(seed, rootKey);
  const [nodes, setNodes, nodesRef] = useCombinedStateAndRef<Base.Node<string>[]>(
    () => {
      const answer = seed == null ? null : combine(seed);
      return answer == null ? [] : toNodes(answer.resources, resolveItem);
    },
  );
  // An unanswered root has no children yet and no empty content: they are the
  // same node list, and only the answer tells them apart.
  const [rootAnswered, setRootAnswered] = useState(
    () => seed != null && combine(seed) != null,
  );
  const loadingListenersRef = useInitializerRef(() => new Set<observe.Handler<void>>());

  const getResourceByKey = (key: string): ontology.Resource | undefined =>
    owners.ownerOf(key)?.get(key) ??
    placeholders.getItem(key) ??
    results?.resources.get(key);

  // A row reads its resource from a source that lists it. When the key moves to another
  // source, the row subscribes to that one.
  const subscribeResource = (
    handler: () => void,
    key: string,
  ): destructor.Destructor => {
    let source: Source | undefined;
    let releaseSource: destructor.Destructor | undefined;
    const bind = () => {
      const owner = owners.ownerOf(key);
      if (owner === source) return;
      releaseSource?.();
      source = owner;
      releaseSource = owner?.subscribe(key, handler);
    };
    bind();
    const releaseOwner = owners.subscribe(() => {
      bind();
      handler();
    }, key);
    const releasePlaceholder = placeholders.subscribe(handler, key);
    return () => {
      releaseSource?.();
      releaseOwner();
      releasePlaceholder();
    };
  };

  // Ends the watch of a node, and the watches of the nodes under it that no other
  // parent lists.
  const dropWatch = (key: string): void => {
    const watch = watchesRef.current.get(key);
    if (watch == null) return;
    watchesRef.current.delete(key);
    watch.release?.();
    owners.notify([...watch.keys].filter((child) => owners.disown(child, key)));
    watch.keys.forEach((child) => {
      if (!owners.has(child)) dropWatch(child);
    });
  };

  // Ends the subscriptions under a hidden node. The answers stay, so a re-expand paints
  // from them.
  const pauseWatch = (key: string): void => {
    const watch = watchesRef.current.get(key);
    if (watch?.release == null) return;
    watch.release();
    watch.release = undefined;
    watch.keys.forEach(pauseWatch);
  };

  const applyAnswer = (parent: resource.ID, watch: Watch): void => {
    const answer = combine(watch);
    if (answer == null) return;
    const parentKey = resource.idToString(parent);
    const previous = watch.keys;
    watch.keys = new Set(answer.sources.keys());
    const left = [...previous].filter((key) => !watch.keys.has(key));
    owners.notify([
      ...[...answer.sources]
        .filter(([key, source]) => owners.own(key, parentKey, source))
        .map(([key]) => key),
      ...left.filter((key) => owners.disown(key, parentKey)),
    ]);
    // A placeholder stands in only until the Core delivers the real resource, which
    // the answer carries.
    const settled = [...watch.keys].filter(placeholders.hasItem);
    if (settled.length > 0) placeholders.deleteItem(settled);
    const order = filterVisible(answer.resources, resolveItem)
      .toSorted(compareResources)
      .map(({ id }) => resource.idToString(id));
    // An answer that moves no node leaves the node list alone, so a change to one
    // resource renders only its own row.
    const moves = watch.order == null || !compare.arraysEqual(order, watch.order);
    watch.order = order;
    // Results keep the order of the matches, so the tree does not sort them, and the
    // children of a group arrive sorted instead.
    const next = toNodes(
      results == null ? answer.resources : answer.resources.toSorted(compareResources),
      resolveItem,
    );
    const nextKeys = new Set(next.map(({ key }) => key));
    // The answer is the authority on its parent's membership. A node it omits
    // survives only while a placeholder backs it, since an optimistic row the Core
    // has not heard about yet cannot be in any answer.
    if (moves)
      setNodes((prevNodes) => {
        const kept = keepLoadedChildren(prevNodes, next);
        const merge = (prevChildren: Base.Node<string>[]): Base.Node<string>[] => [
          ...prevChildren.filter(
            ({ key }) => !nextKeys.has(key) && placeholders.hasItem(key),
          ),
          ...kept,
        ];
        if (parentKey === rootKey) return merge(prevNodes);
        // The watch can outlive the parent's presence in the tree. A missing parent means
        // the answer is stale.
        return updateChildren(prevNodes, parentKey, merge);
      });
    if (parentKey === rootKey) setRootAnswered(true);
    // A node that no answer lists has left the tree.
    left.filter((key) => !owners.has(key)).forEach(dropWatch);
    // An expanded node lists its children again when it comes back from another parent,
    // or when its own parent expands again.
    const expanded = new Set(expandedRef.current);
    watch.keys.forEach((key) => {
      if (!expanded.has(key)) return;
      const child = watchesRef.current.get(key);
      if (child == null ? previous.has(key) : child.release != null) return;
      void watchChildren(resource.parseID(key));
    });
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

  // Subscribes the watch of the node, creating it when the node has none.
  const startWatch = (client: Client, parent: resource.ID, existing?: Watch): Watch => {
    const watch = existing ?? createWatch(listChildren(client, parent));
    watchesRef.current.set(resource.idToString(parent), watch);
    // Subscribe before fetching. A change landing between the two would otherwise
    // reach neither the answer nor the tree.
    const releases = watch.sources.map((source, i) =>
      source.onChange((answer) => {
        watch.answers[i] = answer;
        applyAnswer(parent, watch);
      }),
    );
    watch.release = () => releases.forEach((release) => release());
    // A retained answer paints before the fetch that reconfirms it.
    watch.sources.forEach((source, i) => {
      watch.answers[i] = source.getCached() ?? watch.answers[i];
    });
    applyAnswer(parent, watch);
    return watch;
  };

  const watchChildren = async (parent: resource.ID): Promise<void> => {
    if (client == null) return;
    const key = resource.idToString(parent);
    const existing = watchesRef.current.get(key);
    // A live watch has nothing to wait for, and no fetch will come along to end the
    // wait.
    if (existing?.release != null) {
      if (loadingRef.current === key) setLoading(false);
      return;
    }
    try {
      const { sources } = startWatch(client, parent, existing);
      // Each fetched answer reaches onChange, so only a failure needs the result.
      await Promise.all(sources.map(async (source) => await source.retrieve()));
    } catch (e) {
      handleError(e, "Failed to retrieve resources");
    }
    if (loadingRef.current === key) setLoading(false);
  };

  useEffect(() => {
    if (results == null) void watchChildren(root);
    return () => [...watchesRef.current.keys()].forEach(dropWatch);
  }, [client, rootKey]);

  // Selection drives the context menu, and a key with no node poisons every later
  // right-click, so a node that leaves the tree leaves the selection with it.
  useEffect(() => {
    setSelected((prev) => {
      const next = prev.filter((key) => Base.findNode({ tree: nodes, key }) != null);
      return next.length === prev.length ? prev : next;
    });
  }, [nodes, setSelected]);

  const handleExpand = ({ action, clicked }: Base.HandleExpandProps) => {
    if (action === "contract") return pauseWatch(clicked);
    if (results?.fixed.has(clicked) === true) return;
    setLoading(clicked);
    void watchChildren(resource.parseID(clicked));
  };

  const getResource = ((id: resource.ID | resource.ID[] | string | string[]) => {
    const isSingle = !Array.isArray(id);
    const ids = array.toArray(id);
    const resources = resource
      .idToString(ids)
      .map(getResourceByKey)
      .filter((r) => r != null);
    if (isSingle) {
      if (resources[0] == null)
        throw new NotFoundError(`Resource ${resource.idToString(id)} not found`);
      return resources[0];
    }

    return resources;
  }) as GetResource;

  const getItem = List.createGetItem<string, ontology.Resource>(
    getResourceByKey,
    (keys) => keys.map(getResourceByKey).filter((r) => r != null),
  );

  const sorted = readResources(nodes, getResourceByKey);

  const sort = (a: Base.Node<string>, b: Base.Node<string>) => {
    const aResource = sorted.get(a.key);
    const bResource = sorted.get(b.key);
    if (aResource == null && bResource == null) return 0;
    if (aResource == null) return 1;
    if (bResource == null) return -1;
    return compareResources(aResource, bResource);
  };

  const treeProps = Base.use({
    nodes,
    onExpand: handleExpand,
    selected,
    onSelectedChange: setSelected,
    sort: results == null ? sort : undefined,
  });
  const { shape, expand, contract, expanded } = treeProps;
  const expandedRef = useSyncedRef(expanded);

  // Groups open on arrival once, so a group the user closes stays closed.
  const openedRef = useInitializerRef(() => new Set<string>());
  useEffect(() => {
    if (results == null) return;
    owners.notify([...results.resources.keys()]);
    // A group that now holds only matches drops every child its watch gave.
    [...watchesRef.current.keys()]
      .filter((key) => results.fixed.has(key))
      .forEach(dropWatch);
    const next = keepWatchedChildren(
      nodesRef.current,
      results.nodes,
      watchesRef.current,
      placeholders.hasItem,
    );
    // A group the results leave out loses its watch, so it loads its children again
    // when it comes back.
    [...watchesRef.current.keys()]
      .filter((key) => Base.findNode({ tree: next, key }) == null)
      .forEach(dropWatch);
    setNodes(next);
    // An open group that starts to match gets no expand, so it loads its children here.
    expanded
      .filter(
        (key) =>
          !results.fixed.has(key) &&
          watchesRef.current.get(key)?.release == null &&
          Base.findNode({ tree: results.nodes, key }) != null,
      )
      .forEach((key) => void watchChildren(resource.parseID(key)));
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

  const getBaseProps = (client: Client): BaseProps => ({
    client,
    store,
    openTab,
    addStatus,
    handleError,
  });

  // A result at the top of the tree sits in groups the tree does not show.
  const findAncestors = (tree: Base.Node<string>[], key: string): resource.ID[] => {
    const ancestors: resource.ID[] = [];
    let top = key;
    let parent = Base.findNodeParent({ tree, key });
    while (parent != null) {
      ancestors.push(resource.parseID(parent.key));
      top = parent.key;
      parent = Base.findNodeParent({ tree, key: parent.key });
    }
    ancestors.push(...(results?.paths.get(top) ?? []), root);
    return ancestors;
  };

  const handleDrop = (
    key: string,
    { source, items }: Haul.OnDropProps,
  ): Haul.Item[] => {
    const nodesSnapshot = nodesRef.current;
    const dropped = Base.filterHaulItems(items);
    const isValidDrop = dropped.length > 0 && source.type === Base.HAUL_TYPE;
    if (!isValidDrop || client == null) return [];
    const destination = getResourceByKey(key);
    if (destination == null) return [];
    const { canDrop, drop } = resolveItem(destination.id.type);
    if (drop == null) return [];
    // An item inside another dragged item travels with it. Moving it explicitly
    // would re-parent it onto the destination as a sibling of its own parent.
    const contained = findContainedKeys(
      nodesSnapshot,
      dropped.map(({ key }) => key),
      results?.ancestors,
    );
    const haulMoved = dropped.filter(({ key }) => !contained.has(key));
    const moved = haulMoved.map(({ key }) => ({
      id: resource.parseID(key),
      ancestors: findAncestors(nodesSnapshot, key),
    }));
    const ancestors = findAncestors(nodesSnapshot, key);
    if (moved.length === 0) return [];
    if (!canDrop({ source, items, root, destination, ancestors, moved })) return [];
    contract(...haulMoved.map(({ key }) => key));
    handleError(
      async () => await drop({ client, destination, items: moved }),
      "Failed to move resources",
    );
    return haulMoved;
  };

  const { startDrag, onDragEnd } = Haul.useDrag({ type: Base.HAUL_TYPE });

  const handleDragStart = (itemKey: string) => {
    if (selectedRef.current.includes(itemKey)) {
      const selectedHaulItems = getResource(selectedRef.current).flatMap((res) => {
        const items: Haul.Item[] = [Base.createHaulItem(resource.idToString(res.id))];
        const svcItems = resolveItem(res.id.type).haulItems(res);
        if (svcItems != null) items.push(...svcItems);
        return items;
      });
      return startDrag(selectedHaulItems);
    }
    const haulItems = resolveItem(resource.parseID(itemKey).type).haulItems(
      getResource(itemKey),
    );
    startDrag([Base.createHaulItem(itemKey), ...haulItems]);
  };

  const renderContextMenu = ({ keys: clicked }: Menu.ContextMenuMenuProps) => {
    if (client == null) return <FallbackContextMenu />;
    if (clicked.length === 0) return <RootContextMenu root={root} state={state} />;
    const rightClickedButNotSelected = clicked.find((v) => !selected.includes(v));
    // A right-click outside the selection shows a menu for that item alone. Otherwise
    // the selection state gives the keys, since the virtual tree can lack the DOM
    // elements.
    const keys =
      rightClickedButNotSelected != null ? [rightClickedButNotSelected] : selected;

    const ids = keys.map((key) => resource.parseID(key));

    // A selected node's own children do not count, since an action on the node carries
    // them along. Flat search results show both at the same depth.
    const contained = findContainedKeys(nodes, keys, results?.ancestors);
    const top = Base.getAllNodesOfMinDepth(
      Base.filterShape(shape, (key) => keys.includes(key) && !contained.has(key)),
    );
    const parents = unique.by(
      top.map((key) => findAncestors(nodes, key)[0]),
      (id) => resource.idToString(id),
    );
    const parentID = parents.length === 1 ? parents[0] : null;

    const firstID = ids[0];

    const props: ContextMenuProps = {
      selection: {
        rootID: root,
        parentID,
        ids,
        top: top.map((key) => resource.parseID(key)),
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
        emptyContent={rootAnswered || results != null ? emptyContent : null}
        className={className}
        pinned={pinned}
        onContextMenu={menuProps.open}
        ref={ref}
        onKeyDown={onKeyDown}
      >
        {itemRenderProp}
      </Base.Tree>
    </Context>
  );
};

const LIMIT = 100;
const DEBOUNCE_MS = 120;
const NO_GROUPS: group.Group[] = [];

type FoundMatches = Extract<Matches, { variant: "success" }>;
type FailedMatches = Extract<Matches, { variant: "error" }>;

const isFound = (matches: Matches): matches is FoundMatches =>
  matches.variant === "success";

const isFailed = (matches: Matches): matches is FailedMatches =>
  matches.variant === "error";

const isFolderNode = ({ key }: Base.Node<string>): boolean =>
  FOLDER_TYPES.has(resource.parseID(key).type);

const foldersFirst = (nodes: Base.Node<string>[]): Base.Node<string>[] =>
  [...nodes.filter(isFolderNode), ...nodes.filter((node) => !isFolderNode(node))].map(
    (node) =>
      node.children == null ? node : { ...node, children: foldersFirst(node.children) },
  );

interface BuildResultsParams {
  matches: ontology.Resource[];
  ancestry: Placement;
  layout: SearchLayout;
  hasChildren: (id: resource.ID) => boolean;
}

interface Built {
  results: ResultNodes;
  groups: Map<string, string>;
}

const buildResults = ({
  matches,
  ancestry: { resources, paths },
  layout,
  hasChildren,
}: BuildResultsParams): Built => {
  const names = new Map(resources.map(({ key, name }) => [key, name]));
  const ids = matches.map(({ id }) => id);
  const matched = new Set(resource.idToString(ids));
  const nodes: Base.Node<string>[] = [];
  const above = new Map<string, resource.ID[]>();
  const ancestors = new Map<string, Set<string>>();
  const groups = new Map<string, string>();
  const fixed = new Set<string>();
  const expanded = new Set<string>();
  const toNode = (id: resource.ID): Base.Node<string> => ({
    key: resource.idToString(id),
    children: hasChildren(id) ? [] : undefined,
  });
  ids.forEach((id) => {
    const key = resource.idToString(id);
    const idPaths = paths[key] ?? [];
    // The tree shows only what sits under its root.
    if (idPaths.length === 0) return;
    const parent = idPaths[0].at(-1);
    above.set(key, idPaths[0].toReversed());
    ancestors.set(key, new Set(idPaths.flatMap((path) => resource.idToString(path))));
    if (layout === "flat") {
      const group = parent == null ? undefined : names.get(resource.idToString(parent));
      if (group != null) groups.set(key, group);
      nodes.push(toNode(id));
      return;
    }
    idPaths.forEach((path) => {
      let level = nodes;
      path.forEach((groupID) => {
        const groupKey = resource.idToString(groupID);
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
    results: {
      nodes: foldersFirst(nodes),
      fixed,
      expanded: [...expanded],
      paths: above,
      resources: new Map([...resources, ...matches].map((r) => [r.key, r])),
      ancestors,
    },
    groups,
  };
};

interface Held {
  key: string;
  layout: SearchLayout;
  groups: group.Group[] | null;
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
  useMatches = useGroupMatches,
}: CreateSearchParams): UseSearch => {
  const useSearch = (root: resource.ID | null): Search => {
    const items = useItems();
    const [open, setOpen] = useState(false);
    const [term, setTerm] = useState("");
    const [layout, setLayout] = useState<SearchLayout>("flat");
    const fieldRef = useRef<HTMLInputElement>(null);
    const resultsRef = useRef<HTMLDivElement>(null);
    const treeRef = useRef<HTMLDivElement>(null);
    const searching = open && term.trim() !== "";
    const trimmed = useDebounced(term).trim();
    const sources = useMatches(
      !open || root == null || trimmed === ""
        ? null
        : { term: trimmed, root, limit: LIMIT + 1 },
    );
    const found = sources.every(isFound)
      ? unique.by(
          sources.flatMap(({ resources }) => resources),
          ({ key }) => key,
        )
      : null;
    const shown = found?.slice(0, LIMIT) ?? null;
    const above = shown == null ? [] : groupsAbove(shown);
    const ancestors = Group.useResultAncestors(
      above.length === 0 ? null : { keys: above },
    );
    let groups: group.Group[] | null = null;
    if (above.length === 0) groups = NO_GROUPS;
    else if (ancestors.variant === "success") groups = ancestors.data;
    const key = shown == null ? null : [trimmed, ...shown.map(matchKey)].join("\n");
    const more = found != null && found.length > LIMIT;

    // The last answer shows until the next one is placed, so the results never blank
    // between keystrokes.
    const [held, setHeld] = useState<Held | null>(null);
    if (!searching) {
      if (held != null) setHeld(null);
    } else if (
      root != null &&
      key != null &&
      shown != null &&
      groups != null &&
      (held?.key !== key ||
        held.groups !== groups ||
        held.layout !== layout ||
        held.more !== more)
    ) {
      const { results, groups: groupNames } = buildResults({
        matches: shown,
        ancestry: placeInGroups(shown, groups, root),
        layout,
        hasChildren: (id) => (items[id.type] ?? DefaultItem).hasChildren,
      });
      setHeld({
        key,
        layout,
        groups,
        results,
        match: { term: trimmed, groups: groupNames },
        more,
      });
    }

    const failure =
      sources.find(isFailed) ??
      (ancestors.variant === "error" ? ancestors.status : null);
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
        // A hidden tree cannot take the focus, so the close renders first.
        flushSync(() => {
          setOpen(false);
          setTerm("");
        });
        treeRef.current?.focus({ preventScroll: true });
      },
      setTerm,
      setLayout,
      fieldRef,
      resultsRef,
      treeRef,
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

interface ResultsProps extends Pick<
  Search,
  "noun" | "layout" | "fieldRef" | "resultsRef"
> {
  root: resource.ID;
  answer: SearchAnswer;
}

const Results = ({
  root,
  noun,
  layout,
  answer,
  fieldRef,
  resultsRef,
}: ResultsProps): ReactElement => {
  // A second Escape, in the field, closes the search.
  const handleKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key !== "Escape" || e.target !== e.currentTarget) return;
    e.stopPropagation();
    fieldRef.current?.focus();
  };
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
        ref={resultsRef}
        onKeyDown={handleKeyDown}
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
  root?: resource.ID | null;
  /**
   * Lists the children of the root. Defaults to the children the item of the root's
   * type lists, for a root that is a resource.
   */
  listRoot?: (client: Client) => Source[];
  emptyContent?: ReactNode;
  /** The menu for a right-click on empty space. */
  rootContextMenu?: FC<DefaultContextMenuProps>;
  /** Shows the search answer in place of the tree while the search has one. */
  search?: Search;
}

export const Tree = ({
  root,
  listRoot,
  emptyContent,
  rootContextMenu,
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
          fieldRef={search.fieldRef}
          resultsRef={search.resultsRef}
        />
      )}
      <Internal
        root={root}
        listRoot={listRoot}
        ref={search?.treeRef}
        emptyContent={emptyContent}
        rootContextMenu={rootContextMenu}
        className={search?.answer != null ? CSS.BM("tree", "hidden") : undefined}
      />
    </>
  );
};
