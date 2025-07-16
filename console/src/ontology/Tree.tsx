// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, ontology, type Synnax } from "@synnaxlabs/client";
import {
  Component,
  Haul,
  Icon,
  List,
  Menu,
  Ontology,
  Select,
  Status,
  Synnax as PSynnax,
  Text,
  Tree as Core,
  useAsyncEffect,
  useCombinedStateAndRef,
  useInitializerRef,
  useRequiredContext,
  useSyncedRef,
} from "@synnaxlabs/pluto";
import { array, type observe } from "@synnaxlabs/x";
import { type MutationFunction, useMutation } from "@tanstack/react-query";
import {
  createContext,
  type ReactElement,
  useCallback,
  useMemo,
  useRef,
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
}

interface ContextValue {
  onRename: (key: string, name: string) => void;
  onDrop: (key: string, props: Haul.OnDropProps) => Haul.Item[];
  onDoubleClick: (key: string) => void;
  useLoading: (key: string) => boolean;
}

const Context = createContext<ContextValue | null>(null);

const useContext = (): ContextValue => useRequiredContext(Context);

const DefaultItem = ({
  onDoubleClick,
  resource,
  onDrop,
  onRename,
  icon,
  ...rest
}: TreeItemProps) => (
  <Core.Item {...rest}>
    <Text.Editable level="p" value={resource.name} onChange={onRename} />
  </Core.Item>
);

const itemRenderProp = Component.renderProp((props: Core.ItemProps<string>) => {
  const { itemKey } = props;
  const id = ontology.idZ.parse(itemKey);
  const resource = List.useItem<string, ontology.Resource>(itemKey);
  const selectProps = Select.useItemState<string>(itemKey);
  const service = useServices()[id.type];
  const Item = service.Item ?? DefaultItem;
  const context = useContext();
  const handleRename = useCallback(
    (name: string) => context.onRename(itemKey, name),
    [context, itemKey],
  );
  const handleDrop = useCallback(
    (props: Haul.OnDropProps) => context.onDrop(itemKey, props),
    [context, itemKey],
  );
  const handleDoubleClick = useCallback(
    () => context.onDoubleClick(itemKey),
    [context, itemKey],
  );
  const loading = context.useLoading(itemKey);
  if (resource == null) return null;
  const icon = Icon.resolve(
    typeof service.icon === "function" ? service.icon(resource) : service.icon,
  );
  return (
    <Item
      {...props}
      id={id}
      onDrop={handleDrop}
      onDoubleClick={handleDoubleClick}
      icon={icon as Icon.ReactElement}
      resource={resource}
      loading={loading}
      onRename={handleRename}
      {...selectProps}
    />
  );
});

const Internal = ({ root }: InternalProps): ReactElement => {
  const services = useServices();
  const [selected, setSelected, selectedRef] = useCombinedStateAndRef<string[]>([]);
  const loadingRef = useRef<string | false>(false);
  const [nodes, setNodes, nodesRef] = useCombinedStateAndRef<Core.Node<string>[]>([]);
  const resourcesRef = useInitializerRef(() => new Map<string, ontology.Resource>());
  const listenersRef = useInitializerRef(
    () => new Map<observe.Handler<void>, string>(),
  );
  const loadingListenersRef = useInitializerRef(() => new Set<observe.Handler<void>>());
  const handleError = Status.useErrorHandler();
  const client = PSynnax.use();

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

  const subscribe = useCallback((callback: () => void, key?: string): (() => void) => {
    if (key == null) return () => {};
    listenersRef.current.set(callback, key);
    return () => {
      listenersRef.current.delete(callback);
    };
  }, []);

  const getItem = useCallback((key?: string) => {
    if (key == null) return undefined;
    return resourcesRef.current.get(key);
  }, []);

  useAsyncEffect(
    async (signal) => {
      if (client == null) return;
      const resources = await client.ontology.retrieveChildren(root);
      resources.forEach((r) => resourcesRef.current.set(ontology.idToString(r.id), r));
      if (signal.aborted) return;
      const nodes = resources.map((c) => ({
        key: ontology.idToString(c.id),
        children: [],
      }));
      setNodes(nodes);
    },
    [client, root],
  );

  const handleSyncResourceSet = useCallback(
    (id: ontology.ID) => {
      handleError(async () => {
        if (client == null) return;
        const resource = await client.ontology.retrieve(id);
        console.log(resourcesRef.current.size);
        resourcesRef.current.set(ontology.idToString(id), resource);
      });
    },
    [client, handleError, resourcesRef],
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
    const visibleNode = Core.findNode({
      tree: nodesRef.current,
      key: ontology.idToString(from),
    });
    if (visibleNode == null) return;
    setNodes((prevNodes) => {
      let destination: string | null = ontology.idToString(from);
      if (ontology.idsEqual(from, root)) destination = null;
      return Core.setNode({
        tree: prevNodes,
        destination,
        additions: [{ key: ontology.idToString(to), children: [] }],
      });
    });
  }, []);
  Ontology.useRelationshipSetSynchronizer(handleSyncRelationshipSet);

  const handleExpand = useCallback(
    ({ action, clicked: clickedStringID }: Core.HandleExpandProps<string>): void => {
      if (action !== "expand") return;
      handleError(async () => {
        if (client == null) throw new DisconnectedError();
        if (!resourcesRef.current.has(clickedStringID)) return;
        const clickedID = ontology.idZ.parse(clickedStringID);
        const resources = await client.ontology.retrieveChildren(clickedID);
        resources.forEach((r) =>
          resourcesRef.current.set(ontology.idToString(r.id), r),
        );
        const converted = resources.map((r) => ({
          key: ontology.idToString(r.id),
          children: [],
        }));
        const resourceIDs = new Set(resources.map((r) => ontology.idToString(r.id)));
        setNodes((prevNodes) =>
          Core.updateNodeChildren({
            tree: prevNodes,
            parent: clickedStringID,
            updater: (prevNodes) => [
              ...prevNodes.filter(({ key }) => !resourceIDs.has(key)),
              ...converted,
            ],
          }),
        );
      });
    },
    [],
  );

  const { shape, expand, contract, onSelect } = Core.use({
    nodes,
    onExpand: handleExpand,
    onSelectedChange: setSelected,
  });
  const shapeRef = useSyncedRef(shape);

  const setResource = useCallback(
    (resource: ontology.Resource | ontology.Resource[]) => {
      const resources = array.toArray(resource);
      const resourceIDs = new Set(resources.map((r) => ontology.idToString(r.id)));
      resources.forEach((r) => resourcesRef.current.set(ontology.idToString(r.id), r));
      listenersRef.current.forEach((key, listener) => {
        if (resourceIDs.has(key)) listener();
      });
    },
    [],
  );

  const getResource = useCallback(
    ((id: ontology.ID | ontology.ID[]) => {
      const isSingle = !Array.isArray(id);
      const ids = array.toArray(id);
      const resources = ids.map((id) =>
        resourcesRef.current.get(ontology.idToString(id)),
      );
      if (isSingle) {
        if (resources[0] == null)
          throw new Error(`Resource ${ontology.idToString(id)} not found`);
        return resources[0];
      }
      return resources;
    }) as GetResource,
    [resourcesRef],
  );

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
      setSelection: (keys: string[]) => {
        setNodes(Core.findNodes({ tree: nodesRef.current, keys }));
      },
    }),
    [expand, contract, setLoading, handleError, resourcesRef, nodesRef, setNodes],
  );

  const getBaseProps = useCallback(
    (client: Synnax): BaseProps => ({
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
      setResource(prev);
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
      resourcesRef,
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
    }),
    [handleRename, handleDrop, handleDoubleClick, useLoading],
  );

  console.log(shape);

  return (
    <Context.Provider value={contextValue}>
      <Menu.ContextMenu menu={handleContextMenu} {...menuProps} />
      <Core.Tree<string, ontology.Resource>
        showRules
        shape={shape}
        selected={selected}
        onSelect={onSelect}
        subscribe={subscribe}
        getItem={getItem}
      >
        {itemRenderProp}
      </Core.Tree>
    </Context.Provider>
  );
};

export interface TreeProps {
  root?: ontology.ID | null;
}

export const Tree = ({ root }: TreeProps): ReactElement | null => {
  if (root == null) return null;
  return <Internal root={root} />;
};
