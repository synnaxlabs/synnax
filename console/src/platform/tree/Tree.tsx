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
import {
  Component,
  context,
  Haul,
  List,
  Menu,
  Ontology,
  Status,
  Synnax,
  Tree as Base,
  useCombinedStateAndRef,
  useInitializerRef,
  useSyncedRef,
} from "@synnaxlabs/pluto";
import { array, type destructor, type observe } from "@synnaxlabs/x";
import {
  type DragEvent,
  type ReactElement,
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
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
import { type Action, type State } from "@/session/store";

interface InternalProps {
  root: ontology.ID;
  emptyContent?: ReactNode;
}

interface ContextValue {
  onDrop: (key: string, props: Haul.OnDropProps) => Haul.Item[];
  onDragStart: (itemKey: string) => void;
  onDragEnd: (e: DragEvent) => void;
  useLoading: (key: string) => boolean;
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

const itemRenderProp = Component.renderProp(
  ({ onDrop: _, ...rest }: Base.ItemProps<string>) => {
    const { itemKey } = rest;
    const id = ontology.idZ.parse(itemKey);
    const resource = List.useItem<string, ontology.Resource>(itemKey);
    const Item = useItems()[id.type] ?? DefaultItem;
    const { onDrop, useLoading, onDragStart, onDragEnd } =
      useContext("Tree.itemRenderProp");
    const handleDragStart = useCallback(
      () => onDragStart(itemKey),
      [onDragStart, itemKey],
    );
    const loading = useLoading(itemKey);

    const [draggingOver, setDraggingOver] = useState(false);

    const onDropDrops = Haul.useDrop({
      type: Base.HAUL_TYPE,
      key: itemKey,
      canDrop: useCallback(({ items: entities, source }) => {
        const keys = entities.map((item) => item.key);
        setDraggingOver(false);
        return source.type === Base.HAUL_TYPE && !keys.includes(itemKey);
      }, []),
      onDrop: useCallback((props) => onDrop(itemKey, props) ?? [], [onDrop, itemKey]),
      onDragOver: useCallback(() => setDraggingOver(true), []),
    });

    if (resource == null) return null;

    return (
      <Item
        {...rest}
        draggingOver={draggingOver}
        onDragStart={handleDragStart}
        draggable
        {...onDropDrops}
        onDragLeave={() => setDraggingOver(false)}
        onDragEnd={onDragEnd}
        resource={resource}
        loading={loading}
      />
    );
  },
);

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
  const resolveItem = useCallback(
    (type: ontology.ResourceType): Item => items[type] ?? DefaultItem,
    [items],
  );
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
  // inline rename) before the cluster delivers the real resource.
  const placeholders = List.useMapData<string, ontology.Resource>();

  const getResourceByKey = useCallback(
    (key: string): ontology.Resource | undefined =>
      client?.ontology.cache.resources.get(key) ?? placeholders.getItem(key),
    [client, placeholders],
  );

  const applyAnswer = useCallback(
    (parent: ontology.ID, resources: ontology.Resource[]) => {
      const next = toNodes(resources, resolveItem);
      const nextKeys = new Set(next.map(({ key }) => key));
      // The answer is the authority on its parent's membership. A node it omits
      // survives only while a placeholder backs it, since an optimistic row the
      // cluster has not heard about yet cannot be in any answer.
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
    },
    [resolveItem, setNodes, root, placeholders],
  );

  const useLoading = useCallback(
    (key: string) =>
      useSyncExternalStore<boolean>(
        useCallback((callback) => {
          loadingListenersRef.current.add(callback);
          return () => loadingListenersRef.current.delete(callback);
        }, []),
        useCallback(() => loadingRef.current === key, [key]),
      ),
    [],
  );

  const setLoading = useCallback(
    (key: string | false) => {
      loadingRef.current = key;
      loadingListenersRef.current.forEach((callback) => callback());
    },
    [loadingListenersRef],
  );

  /** Live children subscriptions, keyed by the parent they answer for. */
  const watchedRef = useInitializerRef(() => new Map<string, destructor.Destructor>());

  const watchChildren = useCallback(
    (parent: ontology.ID) => {
      if (client == null) return;
      const key = ontology.idToString(parent);
      if (watchedRef.current.has(key)) return;
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
      handleError(async () => {
        try {
          const resources = await client.ontology.children.retrieve({ ids: parent });
          // A release during the fetch means the answer is no longer wanted.
          if (watchedRef.current.has(key)) applyAnswer(parent, resources);
        } finally {
          if (loadingRef.current === key) setLoading(false);
        }
      }, "Failed to retrieve resources");
    },
    [client, applyAnswer, handleError, setLoading],
  );

  const releaseChildren = useCallback((keys: string[]) => {
    keys.forEach((key) => {
      watchedRef.current.get(key)?.();
      watchedRef.current.delete(key);
    });
  }, []);

  useEffect(() => {
    watchChildren(root);
    return () => releaseChildren([...watchedRef.current.keys()]);
  }, [client, ontology.idToString(root)]);

  const handleSyncResourceSet = useCallback(
    (resource: ontology.Resource) => {
      const hadPlaceholder = placeholders.hasItem(resource.key);
      placeholders.deleteItem(resource.key);
      // Nodes sort by resource name, so an in-tree resource change must rebuild
      // node identity to re-sort. A foreign resource cannot affect this tree.
      if (
        !hadPlaceholder &&
        Base.findNode({ tree: nodesRef.current, key: resource.key }) == null
      )
        return;
      setNodes((prevNodes) => [...prevNodes]);
    },
    [setNodes, nodesRef, placeholders],
  );
  Ontology.useResourceSetSynchronizer(handleSyncResourceSet);
  const handleRelationshipDelete = useCallback(
    (rel: ontology.Relationship) => {
      if (rel.type !== ontology.PARENT_OF_RELATIONSHIP_TYPE) return;
      const removed = ontology.idToString(rel.to);
      const node = Base.findNode({ tree: nodesRef.current, key: removed });
      // Removal and selection cleanup are both no-ops for a node this tree does
      // not hold; return before paying for the tree copy.
      if (node == null) return;
      setNodes((prevNodes) => {
        const parent = ontology.idsEqual(rel.from, root)
          ? null
          : ontology.idToString(rel.from);
        const nextNodes = [
          ...Base.removeNode({
            parent,
            keys: removed,
            tree: Base.deepCopy(prevNodes),
          }),
        ];
        return nextNodes;
      });
      // A deleted node must leave the selection with its subtree. Selection drives
      // the context menu, and a key with no node poisons every later right-click.
      const gone = new Set(
        node == null ? [removed] : Base.getDescendants(node).map(({ key }) => key),
      );
      setSelected((prev) =>
        prev.some((key) => gone.has(key)) ? prev.filter((key) => !gone.has(key)) : prev,
      );
    },
    [setNodes, setSelected, nodesRef, root],
  );
  Ontology.useRelationshipDeleteSynchronizer(handleRelationshipDelete);
  const handleRelationshipSet = useCallback(
    (rel: ontology.Relationship) => {
      if (rel.type !== ontology.PARENT_OF_RELATIONSHIP_TYPE) return;
      const { from, to } = rel;
      setNodes((prevNodes) => {
        let destination: string | null = ontology.idToString(from);
        if (ontology.idsEqual(from, root)) destination = null;
        // setNode no-ops when the destination is not in this tree; keep the
        // previous identity so foreign events do not re-render the tree.
        if (
          destination != null &&
          Base.findNode({ tree: prevNodes, key: destination }) == null
        )
          return prevNodes;
        const tree = Base.deepCopy(prevNodes);
        const key = ontology.idToString(to);
        const existing = Base.findNode({ tree, key });
        const nextNodes = [
          ...Base.setNode({
            tree,
            destination,
            additions: [
              {
                key,
                children:
                  existing?.children ??
                  (resolveItem(to.type).hasChildren ? [] : undefined),
              },
            ],
            throwOnMissing: false,
          }),
        ];
        return nextNodes;
      });
    },
    [setNodes, root, resolveItem],
  );
  Ontology.useRelationshipSetSynchronizer(handleRelationshipSet);

  const handleExpand = useCallback(
    ({ action, clicked }: Base.HandleExpandProps) => {
      if (action === "contract") {
        // A hidden subtree needs no maintenance. Its answers stay cached, so a
        // re-expand paints from them.
        const node = Base.findNode({ tree: nodesRef.current, key: clicked });
        if (node != null)
          releaseChildren(Base.getDescendants(node).map(({ key }) => key));
        return;
      }
      setLoading(clicked);
      watchChildren(ontology.idZ.parse(clicked));
    },
    [watchChildren, releaseChildren, setLoading, nodesRef],
  );

  const getResource = useCallback(
    ((id: ontology.ID | ontology.ID[] | string | string[]) => {
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
    }) as GetResource,
    [getResourceByKey],
  );

  const getItem = useMemo(
    () =>
      List.createGetItem<string, ontology.Resource>(getResourceByKey, (keys) =>
        keys.map(getResourceByKey).filter((r) => r != null),
      ),
    [getResourceByKey],
  );

  const sort = useCallback(
    (a: Base.Node<string>, b: Base.Node<string>) => {
      const aResource = getResourceByKey(a.key);
      const bResource = getResourceByKey(b.key);
      if (aResource == null && bResource == null) return 0;
      if (aResource == null) return 1;
      if (bResource == null) return -1;
      if (aResource.id.type === "group" && bResource.id.type !== "group") return -1;
      if (aResource.id.type !== "group" && bResource.id.type === "group") return 1;
      return aResource.name.localeCompare(bResource.name);
    },
    [getResourceByKey],
  );

  const treeProps = Base.use({
    nodes,
    onExpand: handleExpand,
    selected,
    onSelectedChange: setSelected,
    sort,
  });
  const { shape, expand, contract } = treeProps;
  const shapeRef = useSyncedRef(shape);

  const getState = useCallback(
    (): TreeState => ({
      nodes: nodesRef.current,
      shape: shapeRef.current,
      setNodes,
      expand,
      contract,
      setLoading,
      setResource: placeholders.setItem,
      getResource,
      setSelection: setSelected,
    }),
    [expand, contract, setLoading, handleError, placeholders, nodesRef, setNodes],
  );

  const openTab = Panel.useOpenTab();
  const addStatus = Status.useAdder();
  const store = useStore<State, Action>();

  const moveChildren = Ontology.useMoveChildren({});

  const getBaseProps = useCallback(
    (client: Client): BaseProps => ({
      client,
      store,
      openTab,
      addStatus,
      handleError,
    }),
    [store, openTab, addStatus, handleError],
  );

  const handleDrop = useCallback(
    (key: string, { source, items }: Haul.OnDropProps): Haul.Item[] => {
      const nodesSnapshot = nodesRef.current;
      const dropped = Base.filterHaulItems(items);
      const isValidDrop = dropped.length > 0 && source.type === Base.HAUL_TYPE;
      if (!isValidDrop) return [];
      const destination = ontology.idZ.parse(key);
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
        ids.push(ontology.idZ.parse(key));
        bySource.set(sourceKey, ids);
      });
      contract(...moved.map(({ key }) => key));
      bySource.forEach((ids, sourceKey) =>
        moveChildren.update({
          source: ontology.idZ.parse(sourceKey),
          destination,
          ids,
        }),
      );
      return moved;
    },
    [client, contract, root],
  );

  const { startDrag, onDragEnd } = Haul.useDrag({ type: Base.HAUL_TYPE });

  const handleDragStart = useCallback(
    (itemKey: string) => {
      if (selectedRef.current.includes(itemKey)) {
        const selectedHaulItems = getResource(selectedRef.current).flatMap((res) => {
          const items: Haul.Item[] = [Base.createHaulItem(ontology.idToString(res.id))];
          const svcItems = resolveItem(res.id.type).haulItems(res);
          if (svcItems != null) items.push(...svcItems);
          return items;
        });
        return startDrag(selectedHaulItems);
      }
      const haulItems = resolveItem(ontology.idZ.parse(itemKey).type).haulItems(
        getResource(itemKey),
      );
      startDrag([Base.createHaulItem(itemKey), ...haulItems]);
    },
    [getResource, selectedRef],
  );

  const renderContextMenu = useCallback(
    ({ keys }: Menu.ContextMenuMenuProps) => {
      if (client == null) return <FallbackContextMenu />;
      if (keys.length === 0)
        return <DefaultContextMenu root={root} state={getState()} />;
      const rightClickedButNotSelected = keys.find(
        (v) => !selectedRef.current.includes(v),
      );
      // In the case where we right clicked the menu, but it's not in the current
      // selection, we only display a context menu for that item.
      if (rightClickedButNotSelected != null) keys = [rightClickedButNotSelected];
      // Because we're using a virtualized tree, the keys from the context menu
      // might not actually be accurate (because we're missing DOM elements), so instead
      // we pull directly from the list selected state.
      else keys = selectedRef.current;
      const nodeSnapshot = nodesRef.current;

      const ids = keys.map((key) => ontology.idZ.parse(key));

      // TODO: we might be selecting two nodes that are not ascendants or
      // descendants of the other ones. We need to change this function to
      // implement recursion.
      const parent = Base.findNodeParent({
        tree: nodeSnapshot,
        // We want to find the parent of the node with the lowest depth, since we
        // might be selecting nodes AND their children.
        key: keys.sort(Base.compareDepth(shapeRef.current))[0],
      });

      const parentID = parent == null ? root : ontology.idZ.parse(parent.key);

      const firstID = ontology.idZ.parse(keys[0]);

      const props: ContextMenuProps = {
        selection: {
          rootID: root,
          parentID,
          ids,
        },
        state: getState(),
        ...getBaseProps(client),
      };

      const allSameType = ids.every((id) => id.type === firstID.type);
      if (!allSameType) return <MultipleSelectionContextMenu {...props} />;

      const M = resolveItem(firstID.type).ContextMenu;
      return M == null ? <FallbackContextMenu /> : <M {...props} />;
    },
    [client, setNodes, resolveItem, openTab, nodesRef, setSelected],
  );
  const handleContextMenu = useCallback(
    (props: Menu.ContextMenuMenuProps) => (
      <Errors.SuspenseBoundary loading={null}>
        {renderContextMenu(props)}
      </Errors.SuspenseBoundary>
    ),
    [renderContextMenu],
  );
  const menuProps = Menu.useContextMenu();
  const contextValue = useMemo(
    () => ({
      onDrop: handleDrop,
      useLoading,
      onDragStart: handleDragStart,
      onDragEnd,
    }),
    [handleDrop, handleDragStart, useLoading, onDragEnd],
  );

  return (
    <Context value={contextValue}>
      <Menu.ContextMenu menu={handleContextMenu} {...menuProps} />
      <Base.Tree<string, ontology.Resource>
        {...treeProps}
        showRules
        shape={shape}
        subscribe={placeholders.subscribe}
        // Not getResource: it throws, and a resource may not be cached before
        // the tree attempts to render it.
        getItem={getItem}
        emptyContent={answered ? emptyContent : null}
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
