// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, type Synnax as Client } from "@synnaxlabs/client";
import {
  Component,
  context,
  Flux,
  Haul,
  List,
  Menu,
  Ontology,
  Status,
  Synnax,
  Tree as Base,
  useAsyncEffect,
  useCombinedStateAndRef,
  useInitializerRef,
  useSyncedRef,
} from "@synnaxlabs/pluto";
import { type observe } from "@synnaxlabs/x";
import {
  type DragEvent,
  type ReactElement,
  type ReactNode,
  useCallback,
  useMemo,
  useRef,
  useState,
  useSyncExternalStore,
} from "react";
import { useStore } from "react-redux";

import { Layout } from "@/platform/layout";
import { DefaultContextMenu } from "@/platform/tree/DefaultContextMenu";
import { DefaultItem, type Item } from "@/platform/tree/item";
import { MultipleSelectionContextMenu } from "@/platform/tree/MultipleSelectionContextMenu";
import { useItems } from "@/platform/tree/Provider";
import {
  type BaseProps,
  type ContextMenuProps,
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
  registerName: (key: string, name: string) => void;
}

const [Context, useContext] = context.create<ContextValue>({
  displayName: "Tree.Context",
  providerName: "Tree.Tree",
});

const itemRenderProp = Component.renderProp(
  ({ onDrop: _, ...rest }: Base.ItemProps<string>) => {
    const { itemKey } = rest;
    const id = ontology.idZ.parse(itemKey);
    const resource = List.useItem<string, ontology.ID>(itemKey);
    const Item = useItems()[id.type] ?? DefaultItem;
    const { onDrop, useLoading, onDragStart, onDragEnd, registerName } =
      useContext("Tree.itemRenderProp");
    const handleDragStart = useCallback(
      () => onDragStart(itemKey),
      [onDragStart, itemKey],
    );
    const handleName = useCallback(
      (name: string) => registerName(itemKey, name),
      [registerName, itemKey],
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
        id={id}
        onName={handleName}
        loading={loading}
      />
    );
  },
);

const Internal = ({ root, emptyContent }: InternalProps): ReactElement => {
  const items = useItems();
  const resolveItem = useCallback(
    (type: ontology.ResourceType): Item => items[type] ?? DefaultItem,
    [items],
  );
  const [selected, setSelected, selectedRef] = useCombinedStateAndRef<string[]>([]);
  const loadingRef = useRef<string | false>(false);
  const [nodes, setNodes, nodesRef] = useCombinedStateAndRef<Base.Node<string>[]>([]);
  const fluxStore = Flux.useStore<Ontology.FluxSubStore>();
  const loadingListenersRef = useInitializerRef(() => new Set<observe.Handler<void>>());
  const handleError = Status.useErrorHandler();
  const client = Synnax.use();

  // names caches each item's per-type resolved display name, reported by Items via
  // registerName, so siblings can be sorted alphabetically. Best-effort: a node sorts
  // by key until its name resolves, then nameVersion bumps to re-sort.
  const namesRef = useInitializerRef(() => new Map<string, string>());
  const [nameVersion, setNameVersion] = useState(0);
  const registerName = useCallback((key: string, name: string) => {
    if (namesRef.current.get(key) === name) return;
    namesRef.current.set(key, name);
    setNameVersion((v) => v + 1);
  }, []);

  const retrieveChildren = Ontology.useRetrieveObservableChildren({
    onChange: useCallback(
      ({ data: resources, variant }, { id }) => {
        if (variant == "success") {
          const filtered = resources.filter((r) => {
            const svc = resolveItem(r.type);
            return svc.visible == null || svc.visible(r);
          });
          const converted = filtered.map((r) => ({
            key: ontology.idToString(r),
            children: resolveItem(r.type).hasChildren ? [] : undefined,
          }));
          const ids = new Set(filtered.map((r) => ontology.idToString(r)));
          setNodes((prevNodes) => [
            ...Base.updateNodeChildren({
              tree: prevNodes,
              parent: ontology.idToString(id),
              updater: (prevNodes) => [
                ...prevNodes.filter(({ key }) => !ids.has(key)),
                ...converted,
              ],
            }),
          ]);
        }
        setLoading(false);
      },
      [resolveItem],
    ),
  });

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

  useAsyncEffect(
    async (signal) => {
      if (client == null) return;
      const resources = await client.ontology.retrieveChildren(root);
      if (signal.aborted) return;
      const filtered = resources.filter((r) => {
        const svc = resolveItem(r.type);
        return svc.visible == null || svc.visible(r);
      });
      const nodes = filtered.map((c) => ({
        key: ontology.idToString(c),
        children: resolveItem(c.type).hasChildren ? [] : undefined,
      }));
      setNodes(nodes);
    },
    [client, root],
  );

  const handleRelationshipDelete = useCallback(
    (rel: ontology.Relationship) => {
      if (rel.type !== ontology.PARENT_OF_RELATIONSHIP_TYPE) return;
      setNodes((prevNodes) => {
        const parent = ontology.idsEqual(rel.from, root)
          ? null
          : ontology.idToString(rel.from);
        const nextNodes = [
          ...Base.removeNode({
            parent,
            keys: ontology.idToString(rel.to),
            tree: Base.deepCopy(prevNodes),
          }),
        ];
        return nextNodes;
      });
    },
    [setNodes, parent],
  );
  Ontology.useRelationshipDeleteSynchronizer(handleRelationshipDelete);
  const handleRelationshipSet = useCallback((rel: ontology.Relationship) => {
    if (rel.type !== ontology.PARENT_OF_RELATIONSHIP_TYPE) return;
    const { from, to } = rel;
    setNodes((prevNodes) => {
      let destination: string | null = ontology.idToString(from);
      if (ontology.idsEqual(from, root)) destination = null;
      const nextNodes = [
        ...Base.setNode({
          tree: Base.deepCopy(prevNodes),
          destination,
          additions: [
            {
              key: ontology.idToString(to),
              children: resolveItem(to.type).hasChildren ? [] : undefined,
            },
          ],
          throwOnMissing: false,
        }),
      ];
      return nextNodes;
    });
  }, []);
  Ontology.useRelationshipSetSynchronizer(handleRelationshipSet);

  const handleExpand = useCallback(({ action, clicked }: Base.HandleExpandProps) => {
    if (action !== "expand") return;
    const clickedID = ontology.idZ.parse(clicked);
    setLoading(clicked);
    retrieveChildren.retrieve({ id: clickedID });
  }, []);

  const sort = useCallback(
    (a: Base.Node<string>, b: Base.Node<string>) => {
      const aType = ontology.idZ.parse(a.key).type;
      const bType = ontology.idZ.parse(b.key).type;
      if (aType === "group" && bType !== "group") return -1;
      if (aType !== "group" && bType === "group") return 1;
      const aName = namesRef.current.get(a.key) ?? "";
      const bName = namesRef.current.get(b.key) ?? "";
      return aName.localeCompare(bName);
    },
    // nameVersion re-creates the comparator when a name resolves so the tree re-sorts.
    [nameVersion, namesRef],
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
      setSelection: setSelected,
    }),
    [expand, contract, setLoading, nodesRef, setNodes, setSelected, shapeRef],
  );

  const placeLayout = Layout.usePlacer();
  const removeLayout = Layout.useRemover();
  const addStatus = Status.useAdder();
  const store = useStore<State, Action>();

  const moveChildren = Ontology.useMoveChildren({});

  const getBaseProps = useCallback(
    (client: Client): BaseProps => ({
      client,
      store,
      placeLayout,
      removeLayout,
      addStatus,
      handleError,
    }),
    [store, placeLayout, removeLayout, addStatus, handleError],
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

      const minDepth = Math.min(...dropped.map(({ data }) => data.depth));
      const firstNodeOfMinDepth = dropped.find(({ data }) => data.depth === minDepth);
      if (firstNodeOfMinDepth == null) return [];
      const moved = dropped.filter(({ data }) => data.depth === minDepth);
      const keys = moved.map(({ key }) => key);
      const parent = Base.findNodeParent({
        tree: nodesSnapshot,
        key: firstNodeOfMinDepth.key,
      });
      const sourceID = ontology.idZ.parse(parent?.key ?? ontology.idToString(root));
      contract(...keys);
      const ids = keys.map((key) => ontology.idZ.parse(key));
      moveChildren.update({ source: sourceID, destination, ids });
      return moved;
    },
    [client, contract, root],
  );

  const { startDrag, onDragEnd } = Haul.useDrag({ type: Base.HAUL_TYPE });

  const handleDragStart = useCallback(
    (itemKey: string) => {
      if (selectedRef.current.includes(itemKey)) {
        const selectedHaulItems = selectedRef.current.flatMap((key) => {
          const id = ontology.idZ.parse(key);
          const depth = Base.getDepth(itemKey, shapeRef.current);
          const items: Haul.Item[] = [
            Base.createHaulItem(ontology.idToString(id), depth),
          ];
          const svcItems = resolveItem(id.type).haulItems(id, fluxStore);
          if (svcItems != null) items.push(...svcItems);
          return items;
        });
        return startDrag(selectedHaulItems);
      }
      const depth = Base.getDepth(itemKey, shapeRef.current);
      const id = ontology.idZ.parse(itemKey);
      const haulItems = resolveItem(id.type).haulItems(id, fluxStore);
      startDrag([Base.createHaulItem(itemKey, depth), ...haulItems]);
    },
    [selectedRef, fluxStore, resolveItem, shapeRef, startDrag],
  );

  const handleContextMenu = useCallback(
    ({ keys }: Menu.ContextMenuMenuProps) => {
      if (client == null) return <Layout.DefaultContextMenu />;
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
        key: keys.sort(
          (a, b) =>
            Base.getDepth(a, shapeRef.current) - Base.getDepth(b, shapeRef.current),
        )[0],
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
      return M == null ? <Layout.DefaultContextMenu /> : <M {...props} />;
    },
    [client, setNodes, resolveItem, placeLayout, removeLayout, nodesRef, setSelected],
  );
  const menuProps = Menu.useContextMenu();
  const contextValue = useMemo(
    () => ({
      onDrop: handleDrop,
      useLoading,
      onDragStart: handleDragStart,
      onDragEnd,
      registerName,
    }),
    [handleDrop, handleDragStart, useLoading, onDragEnd, registerName],
  );

  // A resource is fully determined by its node key, so the item is just the parsed id.
  const getItem = useCallback(
    ((key: string | string[]) =>
      Array.isArray(key)
        ? key.map((k) => ontology.idZ.parse(k))
        : ontology.idZ.parse(key)) as List.GetItem<string, ontology.ID>,
    [],
  );

  return (
    <Context value={contextValue}>
      <Menu.ContextMenu menu={handleContextMenu} {...menuProps} />
      <Base.Tree<string, ontology.ID>
        {...treeProps}
        showRules
        shape={shape}
        getItem={getItem}
        emptyContent={emptyContent}
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
