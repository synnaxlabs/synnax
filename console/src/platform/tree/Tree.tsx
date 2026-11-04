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
import { Tree as Base } from "@synnaxlabs/lyra/tree";
import { Access, Ontology, Synnax } from "@synnaxlabs/pluto";
import { array, type destructor, type observe } from "@synnaxlabs/x";
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
import { type Session } from "@/session";

interface InternalProps {
  root: ontology.ID;
  emptyContent?: ReactNode;
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

const Internal = ({ root, emptyContent }: InternalProps): ReactElement => {
  const items = useItems();
  const resolveItem = (type: ontology.ResourceType): Item => items[type] ?? DefaultItem;
  const [selected, setSelected, selectedRef] = useCombinedStateAndRef<string[]>([]);
  const loadingRef = useRef<string | false>(false);
  const handleError = Status.useErrorHandler();
  const client = Synnax.use();
  // The retained answer seeds the first render, not an effect: an effect runs
  // after the commit paints, so the tree would still show one empty frame.
  const [seed] = useState<Base.Node<string>[] | null>(() => {
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
    const merge = (prevChildren: Base.Node<string>[]): Base.Node<string>[] => [
      ...prevChildren.filter(
        ({ key }) => !nextKeys.has(key) && placeholders.hasItem(key),
      ),
      ...keepLoadedChildren(prevChildren, next),
    ];
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
    void watchChildren(root);
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

  const sort = (a: Base.Node<string>, b: Base.Node<string>) => {
    const aResource = getResourceByKey(a.key);
    const bResource = getResourceByKey(b.key);
    if (aResource == null && bResource == null) return 0;
    if (aResource == null) return 1;
    if (bResource == null) return -1;
    if (aResource.id.type === "group" && bResource.id.type !== "group") return -1;
    if (aResource.id.type !== "group" && bResource.id.type === "group") return 1;
    return aResource.name.localeCompare(bResource.name);
  };

  const treeProps = Base.use({
    nodes,
    onExpand: handleExpand,
    selected,
    onSelectedChange: setSelected,
    sort,
  });
  const { shape, expand, contract } = treeProps;
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
  const store = useStore<Session.State, Session.Action>();

  const { update: moveChildren } = Ontology.useMoveChildren({});

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
      const parent = Base.findNodeParent({ tree: nodesSnapshot, key });
      const sourceKey = parent?.key ?? ontology.idToString(root);
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
    const parent = Base.findNodeParent({ tree: nodes, key: shallowest });

    const parentID = parent == null ? root : ontology.parseID(parent.key);

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
        emptyContent={answered ? emptyContent : null}
        pinned={pinned}
        onContextMenu={menuProps.open}
      >
        {itemRenderProp}
      </Base.Tree>
    </Context>
  );
};

export interface TreeProps {
  root?: ontology.ID | null;
  emptyContent?: ReactNode;
}

export const Tree = ({ root, ...rest }: TreeProps): ReactElement | null => {
  if (root == null) return null;
  return <Internal root={root} {...rest} />;
};
