// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  DisconnectedError,
  NotFoundError,
  ontology,
  type Synnax as Client,
} from "@synnaxlabs/client";
import {
  Component,
  Flux,
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
import { array, type observe } from "@synnaxlabs/x";
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
  icon,
  id,
  loading,
  ...rest
}: TreeItemProps) => (
  <Core.Item {...rest} onDoubleClick={onDoubleClick}>
    {icon}
    <Text.MaybeEditable
      id={ontology.idToString(id)}
      value={resource.name}
      onChange
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
    const { onDrop, onDoubleClick, useLoading, onDragStart, onDragEnd } = useContext();
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
        icon={icon}
        resource={resource}
        loading={loading}
      />
    );
  },
);

const Internal = ({ root, emptyContent }: InternalProps): ReactElement => {
  const services = useServices();
  const [selected, setSelected, selectedRef] = useCombinedStateAndRef<string[]>([]);
  const loadingRef = useRef<string | false>(false);
  const [nodes, setNodes, nodesRef] = useCombinedStateAndRef<Core.Node<string>[]>([]);
  const resourceStore = Flux.useStore<Ontology.FluxSubStore>().resources;
  const loadingListenersRef = useInitializerRef(() => new Set<observe.Handler<void>>());
  const handleError = Status.useErrorHandler();
  const client = Synnax.use();

  const retrieveChildren = Ontology.useRetrieveObservableChildren({
    onChange: useCallback(
      ({ data: resources, variant }, { id }) => {
        if (variant == "success") {
          const converted = resources.map((r) => ({
            key: ontology.idToString(r.id),
            children: services[r.id.type].hasChildren ? [] : undefined,
          }));
          const ids = new Set(resources.map((r) => ontology.idToString(r.id)));
          setNodes((prevNodes) => [
            ...Core.updateNodeChildren({
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
      [services],
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
      resources.forEach((r) => resourceStore.set(r));
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
    () => setNodes((prevNodes) => [...prevNodes]),
    [setNodes],
  );
  Ontology.useResourceSetSynchronizer(handleSyncResourceSet);
  const handleRelationshipDelete = useCallback(
    (rel: ontology.Relationship) => {
      if (rel.type !== ontology.PARENT_OF_RELATIONSHIP_TYPE) return;
      setNodes((prevNodes) => {
        const parent = ontology.idsEqual(rel.from, root)
          ? null
          : ontology.idToString(rel.from);
        const nextNodes = [
          ...Core.removeNode({
            parent,
            keys: ontology.idToString(rel.to),
            tree: Core.deepCopy(prevNodes),
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
        ...Core.setNode({
          tree: Core.deepCopy(prevNodes),
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
  Ontology.useRelationshipSetSynchronizer(handleRelationshipSet);

  const handleExpand = useCallback(({ action, clicked }: Core.HandleExpandProps) => {
    if (action !== "expand") return;
    const clickedID = ontology.idZ.parse(clicked);
    setLoading(clicked);
    retrieveChildren.retrieve({ id: clickedID });
  }, []);

  const getResource = useCallback(
    ((id: ontology.ID | ontology.ID[] | string | string[]) => {
      const isSingle = !Array.isArray(id);
      const ids = array.toArray(id);
      const stringIDs = ontology.idToString(ids);
      const resources = resourceStore.get(stringIDs);
      if (isSingle) {
        if (resources[0] == null)
          throw new NotFoundError(`Resource ${ontology.idToString(id)} not found`);
        return resources[0];
      }

      return resources;
    }) as GetResource,
    [resourceStore],
  );

  const subscribe = useCallback(
    (callback: () => void, key: string) => resourceStore.onSet(callback, key),
    [resourceStore],
  );

  const setResource = useCallback(
    (resource: ontology.Resource | ontology.Resource[]) => resourceStore.set(resource),
    [resourceStore],
  );

  const sort = useCallback(
    (a: Core.Node<string>, b: Core.Node<string>) => {
      const [aResource, bResource] = resourceStore.get([a.key, b.key]);
      if (aResource == null && bResource == null) return 0;
      if (aResource == null) return 1;
      if (bResource == null) return -1;
      if (aResource.id.type === "group" && bResource.id.type !== "group") return -1;
      if (aResource.id.type !== "group" && bResource.id.type === "group") return 1;
      return aResource.name.localeCompare(bResource.name);
    },
    [resourceStore],
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
      setResource,
      getResource,
      setSelection: setSelected,
    }),
    [expand, contract, setLoading, handleError, setResource, nodesRef, setNodes],
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

  const moveChildren = Ontology.useMoveChildren({});

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
      const ids = keys.map((key) => ontology.idZ.parse(key));
      moveChildren.update({ source: sourceID, destination, ids });
      return moved;
    },
    [client, contract, root],
  );

  const { startDrag, onDragEnd } = Haul.useDrag({ type: "Tree.Item" });

  const handleDragStart = useCallback(
    (itemKey: string) => {
      const selectedResources = getResource(selectedRef.current);
      if (selectedRef.current.includes(itemKey)) {
        const selectedHaulItems = selectedResources.flatMap((res) => {
          const svcItems = services[res.id.type].haulItems(res);
          const depth = Core.getDepth(itemKey, shapeRef.current);
          const baseItems: Haul.Item[] = [
            {
              type: Core.HAUL_TYPE,
              key: ontology.idToString(res.id),
              data: { depth },
            },
          ];
          if (svcItems != null)
            baseItems.push(
              ...svcItems.map((i) => ({
                ...i,
                data: { ...i.data, depth },
              })),
            );
          return baseItems;
        });
        return startDrag(selectedHaulItems);
      }
      const haulItems = services[ontology.idZ.parse(itemKey).type].haulItems(
        getResource(itemKey),
      );
      const depth = Core.getDepth(itemKey, shapeRef.current);
      startDrag([
        { type: Core.HAUL_TYPE, key: itemKey, data: { depth } },
        ...haulItems.map((item) => ({ ...item, data: { depth } })),
      ]);
    },
    [getResource, selectedRef],
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
        selection: getResource([key]),
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

      const ids = keys.map((key) => ontology.idZ.parse(key));

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
          ids,
        },
        state: getState(),
        ...getBaseProps(client),
      };

      const allSameType = ids.every((id) => id.type === firstID.type);
      if (!allSameType) return <MultipleSelectionContextMenu {...props} />;

      const M = services[firstID.type].TreeContextMenu;
      return M == null ? <Layout.DefaultContextMenu /> : <M {...props} />;
    },
    [client, setNodes, services, placeLayout, removeLayout, nodesRef, setSelected],
  );
  const menuProps = Menu.useContextMenu();
  const contextValue = useMemo(
    () => ({
      onDrop: handleDrop,
      useLoading,
      onDoubleClick: handleDoubleClick,
      onDragStart: handleDragStart,
      onDragEnd,
    }),
    [handleDrop, handleDoubleClick, useLoading, onDragEnd],
  );

  return (
    <Context.Provider value={contextValue}>
      <Menu.ContextMenu menu={handleContextMenu} {...menuProps} />
      <Core.Tree<string, ontology.Resource>
        {...treeProps}
        showRules
        shape={shape}
        subscribe={subscribe}
        // Use resourceStore.get directly instead of getResource because there is
        // a chance that the resource will not be in the store before the tree attempts
        // to render it.
        getItem={resourceStore.get.bind(resourceStore)}
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
