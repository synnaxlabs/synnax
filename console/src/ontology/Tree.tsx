// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, ontology, type Synnax as Client } from "@synnaxlabs/client";
import {
  Component,
  Haul,
  Icon,
  List,
  Menu,
  Ontology,
  Status,
  Synnax,
  Text,
  Tree as Core,
  useAsyncEffect,
  useCombinedStateAndRef,
  useInitializerRef,
  useRequiredContext,
  useSyncedRef,
} from "@synnaxlabs/pluto";
import { array, deep, type observe } from "@synnaxlabs/x";
import { type MutationFunction, useMutation } from "@tanstack/react-query";
import {
  createContext,
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

import { Layout } from "@/layout";
import { MultipleSelectionContextMenu } from "@/ontology/ContextMenu";
import {
  type BaseProps,
  type GetResource,
  type TreeContextMenuProps,
  type TreeItemProps,
  type TreeState,
} from "@/ontology/service";
import { useServices } from "@/ontology/ServicesProvider";
import { type RootAction, type RootState } from "@/store";

interface InternalProps {
  root: ontology.ID;
  emptyContent?: ReactNode;
}

interface ContextValue {
  onRename: (key: string, name: string) => void;
  onDrop: (key: string, props: Haul.OnDropProps) => Haul.Item[];
  onDragStart: (itemKey: string) => void;
  onDragEnd: (e: DragEvent) => void;
  onDoubleClick: (key: string) => void;
  useLoading: (key: string) => boolean;
}

const Context = createContext<ContextValue | null>(null);

const useContext = (): ContextValue => useRequiredContext(Context);

const DefaultItem = ({
  onDoubleClick,
  resource,
  onRename,
  icon,
  id,
  loading,
  ...rest
}: TreeItemProps) => (
  <Core.Item {...rest} onDoubleClick={onDoubleClick}>
    {icon}
    <Text.Editable
      id={ontology.idToString(id)}
      value={resource.name}
      onChange={onRename}
      allowDoubleClick={false}
      style={{
        userSelect: "none",
        width: 0,
        flexGrow: 1,
      }}
      overflow="ellipsis"
    />
  </Core.Item>
);

const itemRenderProp = Component.renderProp(
  ({ onDrop: _, ...rest }: Core.ItemProps<string>) => {
    const { itemKey } = rest;
    const id = ontology.idZ.parse(itemKey);
    const resource = List.useItem<string, ontology.Resource>(itemKey);
    const service = useServices()[id.type];
    const Item = service.Item ?? DefaultItem;
    const { onRename, onDrop, onDoubleClick, useLoading, onDragStart, onDragEnd } =
      useContext();
    const handleRename = useCallback(
      (name: string) => onRename(itemKey, name),
      [onRename, itemKey],
    );
    const handleDoubleClick = useCallback(
      () => onDoubleClick(itemKey),
      [onDoubleClick, itemKey],
    );
    const handleDragStart = useCallback(
      () => onDragStart(itemKey),
      [onDragStart, itemKey],
    );
    const loading = useLoading(itemKey);

    const [draggingOver, setDraggingOver] = useState(false);

    const onDropDrops = Haul.useDrop({
      type: "Tree.Item",
      key: itemKey,
      canDrop: useCallback(({ items: entities, source }) => {
        const keys = entities.map((item) => item.key);
        setDraggingOver(false);
        return source.type === "Tree.Item" && !keys.includes(itemKey);
      }, []),
      onDrop: useCallback((props) => onDrop(itemKey, props) ?? [], [onDrop, itemKey]),
      onDragOver: useCallback(() => setDraggingOver(true), []),
    });

    if (resource == null) return null;
    const icon = Icon.resolve(
      typeof service.icon === "function" ? service.icon(resource) : service.icon,
    );

    return (
      <Item
        {...rest}
        draggingOver={draggingOver}
        onDragStart={handleDragStart}
        draggable
        id={id}
        {...onDropDrops}
        onDragLeave={() => setDraggingOver(false)}
        onDragEnd={onDragEnd}
        onDoubleClick={handleDoubleClick}
        icon={icon as Icon.ReactElement}
        resource={resource}
        loading={loading}
        onRename={handleRename}
      />
    );
  },
);

const Internal = ({ root, emptyContent }: InternalProps): ReactElement => {
  const services = useServices();
  const [selected, setSelected, selectedRef] = useCombinedStateAndRef<string[]>([]);
  const loadingRef = useRef<string | false>(false);
  const [nodes, setNodes, nodesRef] = useCombinedStateAndRef<Core.Node<string>[]>([]);
  const resourceStore = List.useMapData<string, ontology.Resource>();
  const loadingListenersRef = useInitializerRef(() => new Set<observe.Handler<void>>());
  const handleError = Status.useErrorHandler();
  const client = Synnax.use();

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
      resources.forEach((r) => resourceStore.setItem(r));
      if (signal.aborted) return;
      const nodes = resources.map((c) => ({
        key: ontology.idToString(c.id),
        children: services[c.id.type].hasChildren ? [] : undefined,
      }));
      setNodes(nodes);
    },
    [client, root],
  );

  const handleSyncResourceSet = useCallback(
    (resource: ontology.Resource) => {
      const prev = resourceStore.getItem(ontology.idToString(resource.id));
      resourceStore.setItem(resource);
      // Trigger re-sort when name changes.
      if (prev?.name !== resource.name) setNodes((prevNodes) => [...prevNodes]);
    },
    [client, handleError, resourceStore.setItem],
  );
  Ontology.useResourceSetSynchronizer(handleSyncResourceSet);
  const handleSyncRelationshipDelete = useCallback((rel: ontology.Relationship) => {
    if (rel.type !== ontology.PARENT_OF_RELATIONSHIP_TYPE) return;
    setNodes((prevNodes) =>
      Core.removeNode({ keys: [ontology.idToString(rel.to)], tree: prevNodes }),
    );
  }, []);
  Ontology.useRelationshipDeleteSynchronizer(handleSyncRelationshipDelete);
  const handleSyncRelationshipSet = useCallback((rel: ontology.Relationship) => {
    if (rel.type !== ontology.PARENT_OF_RELATIONSHIP_TYPE) return;
    const { from, to } = rel;
    setNodes((prevNodes) => {
      let destination: string | null = ontology.idToString(from);
      if (ontology.idsEqual(from, root)) destination = null;
      const nextNodes = [
        ...Core.setNode({
          tree: prevNodes,
          destination,
          additions: [
            {
              key: ontology.idToString(to),
              children: services[to.type].hasChildren ? [] : undefined,
            },
          ],
          throwOnMissing: false,
        }),
      ];
      return nextNodes;
    });
  }, []);
  Ontology.useRelationshipSetSynchronizer(handleSyncRelationshipSet);

  const handleExpand = useCallback(
    ({ action, clicked: clickedStringID }: Core.HandleExpandProps<string>): void => {
      if (action !== "expand") return;
      handleError(async () => {
        if (client == null) throw new DisconnectedError();
        if (!resourceStore.hasItem(clickedStringID)) return;
        const clickedID = ontology.idZ.parse(clickedStringID);
        setLoading(clickedStringID);
        const resources = await client.ontology.retrieveChildren(clickedID);
        resources.forEach((r) => resourceStore.setItem(r));
        const converted = resources.map((r) => ({
          key: ontology.idToString(r.id),
          children: services[r.id.type].hasChildren ? [] : undefined,
        }));
        const resourceIDs = new Set(resources.map((r) => ontology.idToString(r.id)));
        setNodes((prevNodes) => [
          ...Core.updateNodeChildren({
            tree: prevNodes,
            parent: clickedStringID,
            updater: (prevNodes) => [
              ...prevNodes.filter(({ key }) => !resourceIDs.has(key)),
              ...converted,
            ],
          }),
        ]);
        setLoading(false);
      });
    },
    [],
  );

  const getResource = useCallback(
    ((id: ontology.ID | ontology.ID[]) => {
      const isSingle = !Array.isArray(id);
      const ids = array.toArray(id);
      const resources = resourceStore.getItem(ids.map((id) => ontology.idToString(id)));
      if (isSingle) {
        if (resources[0] == null)
          throw new Error(`Resource ${ontology.idToString(id)} not found`);
        return resources[0];
      }
      return resources;
    }) as GetResource,
    [resourceStore.getItem],
  );

  const sort = useCallback(
    (a: Core.Node<string>, b: Core.Node<string>) => {
      const [aResource] = getResource([ontology.idZ.parse(a.key)]);
      const [bResource] = getResource([ontology.idZ.parse(b.key)]);
      if (aResource == null && bResource == null) return 0;
      if (aResource == null) return 1;
      if (bResource == null) return -1;
      if (aResource.id.type === "group" && bResource.id.type !== "group") return -1;
      if (aResource.id.type !== "group" && bResource.id.type === "group") return 1;
      return aResource.name.localeCompare(bResource.name);
    },
    [getResource],
  );

  const treeProps = Core.use({
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
      setResource: resourceStore.setItem,
      getResource,
      setSelection: setSelected,
    }),
    [
      expand,
      contract,
      setLoading,
      handleError,
      resourceStore.setItem,
      nodesRef,
      setNodes,
    ],
  );

  const getBaseProps = useCallback(
    (client: Client): BaseProps => ({
      client,
      store,
      placeLayout,
      removeLayout,
      addStatus,
      handleError,
      services,
    }),
    [],
  );

  const placeLayout = Layout.usePlacer();
  const removeLayout = Layout.useRemover();
  const addStatus = Status.useAdder();
  const store = useStore<RootState, RootAction>();

  const rename = useMutation<
    void,
    Error,
    { key: string; name: string },
    { prevName: string }
  >({
    onMutate: ({ key, name: newName }) => {
      const id = ontology.idZ.parse(key);
      const svc = services[id.type];
      if (svc.allowRename == null || svc.onRename == null || client == null) return;
      const state = getState();
      const prevName = state.getResource(id).name;
      svc.onRename?.eager?.({ id, name: newName, state, ...getBaseProps(client) });
      const prev = state.getResource(id);
      prev.name = newName;
      resourceStore.setItem(prev);
      return { prevName };
    },
    mutationFn: useCallback<MutationFunction<void, { key: string; name: string }>>(
      async ({ key, name }: { key: string; name: string }) => {
        const id = ontology.idZ.parse(key);
        const svc = services[id.type];
        if (svc.allowRename == null || svc.onRename == null || client == null) return;
        await svc?.onRename?.execute?.({
          id,
          name,
          state: getState(),
          ...getBaseProps(client),
        });
      },
      [services],
    ),
    onError: (error, { key, name }, ctx) => {
      if (ctx == null || client == null) return;
      const { prevName } = ctx;
      const id = ontology.idZ.parse(key);
      const svc = services[id.type];
      handleError(error, `Failed to rename ${prevName} to ${name}`);
      svc.onRename?.rollback?.(
        { id, name, state: getState(), ...getBaseProps(client) },
        prevName,
      );
    },
  });

  const dropMutation = useMutation<
    void,
    Error,
    { source: ontology.ID; ids: ontology.ID[]; destination: ontology.ID },
    Core.Node<string>[]
  >({
    onMutate: ({ ids, destination }) => {
      const nodesSnapshot = nodesRef.current;
      const prevNodes = Core.deepCopy(nodesSnapshot);
      const keys = ids.map((id) => ontology.idToString(id));
      // Move the nodes in the tree.
      const next = Core.moveNode({
        tree: nodesSnapshot,
        destination: ontology.idToString(destination),
        keys,
      });
      setNodes([...next]);
      return prevNodes;
    },
    mutationFn: async ({ source, ids, destination }) => {
      if (client == null) return;
      await client.ontology.moveChildren(source, destination, ...ids);
    },
    onError: (error, _, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      handleError(error, "Failed to move resources");
    },
  });

  const handleDrop = useCallback(
    (key: string, { source, items }: Haul.OnDropProps): Haul.Item[] => {
      const nodesSnapshot = nodesRef.current;
      const dropped = Haul.filterByType(Core.HAUL_TYPE, items);
      const isValidDrop = dropped.length > 0 && source.type === "Tree.Item";
      if (!isValidDrop) return [];
      const destination = ontology.idZ.parse(key);
      const svc = services[destination.type];
      if (!svc.canDrop({ source, items })) return [];
      const minDepth = Math.min(
        ...dropped.map(({ data }) => (data?.depth ?? 0) as number),
      );
      const firstNodeOfMinDepth = dropped.find(({ data }) => data?.depth === minDepth);
      if (firstNodeOfMinDepth == null) return [];
      const moved = dropped.filter(({ data }) => data?.depth === minDepth);
      const keys = moved.map(({ key }) => key as string);
      const parent = Core.findNodeParent({
        tree: nodesSnapshot,
        key: firstNodeOfMinDepth.key.toString(),
      });
      const sourceID = ontology.idZ.parse(parent?.key ?? ontology.idToString(root));
      contract(...keys);
      dropMutation.mutate({
        source: sourceID,
        ids: keys.map((key) => ontology.idZ.parse(key)),
        destination,
      });
      return moved;
    },
    [client, contract, root],
  );

  const { startDrag, onDragEnd } = Haul.useDrag({
    type: "Tree.Item",
  });

  const handleDragStart = useCallback(
    (itemKey: string) => {
      const selectedResources = getResource(ontology.parseIDs(selectedRef.current));
      if (selectedRef.current.includes(itemKey)) {
        const selectedHaulItems = selectedResources.flatMap((res) => {
          const haulItems = services[res.id.type].haulItems(res);
          const depth = Core.getDepth(itemKey, shapeRef.current);
          return [
            {
              type: Core.HAUL_TYPE,
              key: ontology.idToString(res.id),
              data: { depth },
            },
            ...(haulItems?.map((item) => ({
              ...item,
              data: { ...item.data, depth },
            })) ?? []),
          ];
        });
        return startDrag(selectedHaulItems);
      }
      const haulItems = services[ontology.idZ.parse(itemKey).type].haulItems(
        getResource(ontology.idZ.parse(itemKey)),
      );
      const depth = Core.getDepth(itemKey, shapeRef.current);
      startDrag([
        { type: Core.HAUL_TYPE, key: itemKey, data: { depth } },
        ...haulItems.map((item) => ({ ...item, data: { depth } })),
      ]);
    },
    [getResource, selectedRef],
  );

  const handleRename = useCallback(
    (key: string, name: string) => rename.mutate({ key, name }),
    [rename],
  );

  const handleDoubleClick = useCallback(
    (key: string) => {
      if (client == null) throw new DisconnectedError();
      const { type } = ontology.idZ.parse(key);
      services[type].onSelect?.({
        client,
        store,
        services,
        placeLayout,
        handleError,
        removeLayout,
        addStatus,
        selection: [getResource(ontology.idZ.parse(key))],
      });
    },
    [client, store, services, placeLayout, handleError, removeLayout, addStatus],
  );

  const handleContextMenu = useCallback(
    ({ keys }: Menu.ContextMenuMenuProps) => {
      if (keys.length === 0 || client == null) return <Layout.DefaultContextMenu />;
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

      const resourceIDs = keys.map((key) => ontology.idZ.parse(key));

      // TODO: we might be selecting two nodes that are not ascendants or
      // descendants of the other ones. We need to change this function to
      // implement recursion.
      const parent = Core.findNodeParent({
        tree: nodeSnapshot,
        // We want to find the parent of the node with the lowest depth, since we
        // might be selecting nodes AND their children.
        key: keys.sort((a, b) => Core.getDepth(a, shape) - Core.getDepth(b, shape))[0],
      });

      const parentID = parent == null ? root : ontology.idZ.parse(parent.key);

      const firstID = ontology.idZ.parse(keys[0]);

      const props: TreeContextMenuProps = {
        selection: {
          rootID: root,
          parentID,
          resourceIDs,
        },
        state: getState(),
        ...getBaseProps(client),
      };

      const allSameType = resourceIDs.every((id) => id.type === firstID.type);
      if (!allSameType) return <MultipleSelectionContextMenu {...props} />;

      const M = services[firstID.type].TreeContextMenu;
      return M == null ? <Layout.DefaultContextMenu /> : <M {...props} />;
    },
    [
      client,
      setNodes,
      services,
      placeLayout,
      removeLayout,
      resourceStore.getItem,
      nodesRef,
      setSelected,
    ],
  );
  const menuProps = Menu.useContextMenu();
  const contextValue = useMemo(
    () => ({
      onRename: handleRename,
      onDrop: handleDrop,
      useLoading,
      onDoubleClick: handleDoubleClick,
      onDragStart: handleDragStart,
      onDragEnd,
    }),
    [handleRename, handleDrop, handleDoubleClick, useLoading, onDragEnd],
  );

  return (
    <Context.Provider value={contextValue}>
      <Menu.ContextMenu menu={handleContextMenu} {...menuProps} />
      <Core.Tree<string, ontology.Resource>
        {...treeProps}
        showRules
        shape={deep.copy(shape)}
        subscribe={resourceStore.subscribe}
        getItem={resourceStore.getItem}
        emptyContent={emptyContent}
        onContextMenu={menuProps.open}
      >
        {itemRenderProp}
      </Core.Tree>
    </Context.Provider>
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
