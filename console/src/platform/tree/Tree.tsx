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
import { Tree as Base } from "@synnaxlabs/lyra/tree";
import { Access, Synnax } from "@synnaxlabs/pluto";
import { array, compare, type destructor, type observe } from "@synnaxlabs/x";
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
import { Errors } from "@/platform/errors";
import { Panel } from "@/platform/panel";
import { type Source } from "@/platform/tree/children";
import { DefaultContextMenu } from "@/platform/tree/DefaultContextMenu";
import { DefaultItem, type Item } from "@/platform/tree/item";
import { MultipleSelectionContextMenu } from "@/platform/tree/MultipleSelectionContextMenu";
import { useItems } from "@/platform/tree/Provider";
import {
  type BaseProps,
  type ContextMenuProps,
  type GetResource,
  type TreeState,
} from "@/platform/tree/types";
import { useResources } from "@/platform/tree/useResource";
import { type Action, type State } from "@/session/store";

interface InternalProps extends Omit<TreeProps, "root"> {
  root: resource.ID;
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

const toNodes = (
  resources: ontology.Resource[],
  resolveItem: (type: resource.Type) => Item,
): Base.Node<string>[] =>
  filterVisible(resources, resolveItem).map((r) => ({
    key: resource.idToString(r.id),
    children: resolveItem(r.id.type).hasChildren ? [] : undefined,
  }));

// Groups come first, then resources by name.
const compareResources = (a: ontology.Resource, b: ontology.Resource): number => {
  if (a.id.type === "group" && b.id.type !== "group") return -1;
  if (a.id.type !== "group" && b.id.type === "group") return 1;
  return a.name.localeCompare(b.name);
};

const Internal = ({ root, listRoot, emptyContent }: InternalProps): ReactElement => {
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
    client == null ? null : createWatch(listChildren(client, root)),
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
    owners.ownerOf(key)?.get(key) ?? placeholders.getItem(key);

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
    const next = toNodes(answer.resources, resolveItem);
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
    void watchChildren(root);
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
    sort,
  });
  const { shape, expand, contract, expanded } = treeProps;
  const expandedRef = useSyncedRef(expanded);
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

  const findAncestors = (tree: Base.Node<string>[], key: string): resource.ID[] => {
    const ancestors: resource.ID[] = [];
    let parent = Base.findNodeParent({ tree, key });
    while (parent != null) {
      ancestors.push(resource.parseID(parent.key));
      parent = Base.findNodeParent({ tree, key: parent.key });
    }
    ancestors.push(root);
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

    const ids = keys.map((key) => resource.parseID(key));

    // We want to find the parent of the node with the lowest depth, since we
    // might be selecting nodes AND their children.
    const [shallowest] = keys.toSorted(Base.compareDepth(shape));
    // TODO: we might be selecting two nodes that are not ascendants or descendants of
    // the other ones. We need to change this function to implement recursion.
    const parent = Base.findNodeParent({ tree: nodes, key: shallowest });

    const parentID = parent == null ? root : resource.parseID(parent.key);

    const firstID = resource.parseID(shallowest);

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
        emptyContent={rootAnswered ? emptyContent : null}
        pinned={pinned}
        onContextMenu={menuProps.open}
      >
        {itemRenderProp}
      </Base.Tree>
    </Context>
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
}

export const Tree = ({ root, ...rest }: TreeProps): ReactElement | null => {
  if (root == null) return null;
  return <Internal root={root} {...rest} />;
};
