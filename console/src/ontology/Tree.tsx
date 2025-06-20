// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, ontology } from "@synnaxlabs/client";
import {
  Haul,
  Menu,
  Ontology,
  Status,
  Synnax,
  Tree as Core,
  useAsyncEffect,
  useCombinedStateAndRef,
  useStateRef,
} from "@synnaxlabs/pluto";
import { type MutationFunction, useMutation } from "@tanstack/react-query";
import { memo, type ReactElement, useCallback, useMemo, useState } from "react";
import { useStore } from "react-redux";

import { NULL_CLIENT_ERROR } from "@/errors";
import { Layout } from "@/layout";
import { MultipleSelectionContextMenu } from "@/ontology/ContextMenu";
import {
  type BaseProps,
  type HandleTreeRenameProps,
  type Services,
  type TreeContextMenuProps,
} from "@/ontology/service";
import { useServices } from "@/ontology/ServicesProvider";
import { toTreeNode, toTreeNodes } from "@/ontology/toTreeNode";
import { type RootAction, type RootState } from "@/store";

const sortFunc = (a: Core.Node, b: Core.Node) => {
  const aIsGroup = a.key.startsWith(group.ONTOLOGY_TYPE);
  const bIsGroup = b.key.startsWith(group.ONTOLOGY_TYPE);
  if (aIsGroup && !bIsGroup) return -1;
  if (!aIsGroup && bIsGroup) return 1;
  return Core.defaultSort(a, b);
};

interface InternalProps {
  root: ontology.ID;
}

const Internal = ({ root }: InternalProps): ReactElement => {
  const client = Synnax.use();
  const services = useServices();
  const store = useStore<RootState, RootAction>();
  const placeLayout = Layout.usePlacer();
  const removeLayout = Layout.useRemover();
  const [loading, setLoading] = useState<string | false>(false);
  const [nodes, setNodes, nodesRef] = useCombinedStateAndRef<Core.Node[]>([]);
  const [resourcesRef, setResources] = useStateRef<ontology.Resource[]>([]);
  const [selected, setSelected, selectedRef] = useCombinedStateAndRef<string[]>([]);
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  const menuProps = Menu.useContextMenu();

  const baseProps: BaseProps = useMemo<BaseProps>(() => {
    if (client == null) throw NULL_CLIENT_ERROR;
    return {
      client,
      store,
      placeLayout,
      removeLayout,
      services,
      addStatus,
      handleError,
    };
  }, [client, store, placeLayout, removeLayout, services, addStatus, handleError]);

  useAsyncEffect(
    async (signal) => {
      if (client == null) {
        setNodes([]);
        setResources([]);
        return;
      }
      const children = await client.ontology.retrieveChildren(root);
      if (signal.aborted) return;
      setNodes(toTreeNodes(services, children));
      setResources(children);
    },
    [client, root, services],
  );

  const handleResourceSet = useCallback(
    (id: ontology.ID) => {
      handleError(async () => {
        if (client == null) throw NULL_CLIENT_ERROR;
        const resource = await client.ontology.retrieve(id);
        setResources((prevResources) => {
          const existingIndex = prevResources.findIndex(
            ({ id }) => id.toString() === resource.id.toString(),
          );
          if (existingIndex === -1) return prevResources;
          const nextResources = [...prevResources];
          nextResources[existingIndex] = resource;
          return nextResources;
        });
        setNodes((prevNodes) => {
          const existingIndex = prevNodes.findIndex(
            ({ key }) => key === resource.id.toString(),
          );
          if (existingIndex === -1) return prevNodes;
          const nextNodes = [...prevNodes];
          nextNodes[existingIndex] = toTreeNode(services, resource);
          return nextNodes;
        });
      });
    },
    [client, services, handleError],
  );
  Ontology.useResourceSetSynchronizer(handleResourceSet);

  const handleResourceDelete = useCallback((id: ontology.ID) => {
    setResources((prevResources) =>
      prevResources.filter(({ id }) => id.toString() !== id.toString()),
    );
    setNodes((prevNodes) => prevNodes.filter(({ key }) => key !== id.toString()));
  }, []);
  Ontology.useResourceDeleteSynchronizer(handleResourceDelete);

  const handleRelationshipDelete = useCallback((rel: ontology.Relationship) => {
    if (rel.type !== ontology.PARENT_RELATIONSHIP_TYPE) return;
    setNodes((prevNodes) =>
      Core.removeNode({ keys: [rel.to.toString()], tree: prevNodes }),
    );
  }, []);
  Ontology.useRelationshipDeleteSynchronizer(handleRelationshipDelete);

  const handleRelationshipSet = useCallback((rel: ontology.Relationship) => {
    if (rel.type !== ontology.PARENT_RELATIONSHIP_TYPE) return;
    const { from, to } = rel;
    const visibleNode = Core.findNode({
      tree: nodesRef.current,
      key: from.toString(),
    });
    if (visibleNode == null) return;
    handleError(async () => {
      if (client == null) throw NULL_CLIENT_ERROR;
      const resource = await client.ontology.retrieve(to);
      setResources((prevResources) => {
        const existingIndex = prevResources.findIndex(
          ({ id }) => id.toString() === to.toString(),
        );
        if (existingIndex === -1) return [...prevResources, resource];
        const nextResources = [...prevResources];
        nextResources[existingIndex] = resource;
        return nextResources;
      });
      setNodes((prevNodes) => {
        let destination: string | null = from.toString();
        if (destination === root.toString()) destination = null;
        return Core.setNode({
          tree: prevNodes,
          destination,
          additions: toTreeNodes(services, [resource]),
        });
      });
    });
  }, []);
  Ontology.useRelationshipSetSynchronizer(handleRelationshipSet);

  const handleExpand = useCallback(
    ({ action, clicked }: Core.HandleExpandProps): void => {
      if (action !== "expand") return;
      handleError(async () => {
        if (client == null) throw NULL_CLIENT_ERROR;
        const id = new ontology.ID(clicked);
        try {
          setLoading(clicked);
          if (!resourcesRef.current.find(({ id }) => id.toString() === clicked))
            // This happens when we need add an item to the tree before we create it in
            // the ontology service. For instance, creating a new group will create a
            // new node in the tree, but if onExpand is called before the group is
            // created on the server, an error will be thrown when we try to retrieve
            // the children of the new group.
            return;
          const resources = await client.ontology.retrieveChildren(id);
          const converted = toTreeNodes(services, resources);
          const nextTree = Core.updateNodeChildren({
            tree: nodesRef.current,
            parent: clicked,
            updater: (nodes) => {
              const res = converted.map((node) => {
                const existing = nodes.find(({ key }) => key === node.key);
                return { ...existing, ...node };
              });
              const nodesBeingRenamed = nodes.filter(
                ({ key, name }) =>
                  !converted.find(({ key: k }) => k === key) && name.length === 0,
              );
              return [...res, ...nodesBeingRenamed];
            },
          });
          const keys = resources.map(({ id }) => id.toString());
          resourcesRef.current = [
            // Dedupe any resources that already exist.
            ...resourcesRef.current.filter(({ id }) => !keys.includes(id.toString())),
            ...resources,
          ];
          setNodes([...nextTree]);
        } finally {
          setLoading(false);
        }
      }, "Failed to expand resources tree");
    },
    [client, services],
  );

  const treeProps = Core.use({
    onExpand: handleExpand,
    nodes,
    selected,
    onSelectedChange: setSelected,
    sort: sortFunc,
  });

  const dropMutation = useMutation<
    void,
    Error,
    { source: ontology.ID; ids: ontology.ID[]; destination: ontology.ID },
    Core.Node[]
  >({
    onMutate: ({ ids, destination }) => {
      const nodesSnapshot = nodesRef.current;
      const prevNodes = Core.deepCopy(nodesSnapshot);
      const keys = ids.map((id) => id.toString());
      // Move the nodes in the tree.
      const next = Core.moveNode({
        tree: nodesSnapshot,
        destination: destination.toString(),
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

  const handleDrop: Core.TreeProps["onDrop"] = useCallback(
    (key: string, { source, items }: Haul.OnDropProps): Haul.Item[] => {
      const nodesSnapshot = nodesRef.current;
      const dropped = Haul.filterByType(Core.HAUL_TYPE, items);
      const isValidDrop = dropped.length > 0 && source.type === "Tree.Item";
      if (!isValidDrop) return [];
      const destination = new ontology.ID(key);
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
      const sourceID = new ontology.ID(parent?.key ?? root.toString());
      treeProps.contract(...keys);
      dropMutation.mutate({
        source: sourceID,
        ids: keys.map((key) => new ontology.ID(key)),
        destination,
      });
      return moved;
    },
    [client, treeProps.contract, root],
  );

  const getRenameProps = useCallback(
    (key: string, name: string): HandleTreeRenameProps => {
      const id = new ontology.ID(key);
      return {
        id,
        name,
        state: {
          nodes: nodesRef.current,
          resources: resourcesRef.current,
          setNodes,
          setResources,
        },
        ...baseProps,
      };
    },
    [baseProps, nodesRef, resourcesRef],
  );

  const rename = useMutation<
    void,
    Error,
    { key: string; name: string },
    { prevName: string }
  >({
    onMutate: ({ key, name }) => {
      const rProps = getRenameProps(key, name);
      const svc = services[rProps.id.type];
      if (svc.allowRename == null || svc.onRename == null) return;
      let prevName = "";
      const nodes = Core.updateNode({
        tree: nodesRef.current,
        key,
        updater: (node) => {
          prevName = node.name;
          return { ...node, name };
        },
      });
      setNodes([...nodes]);
      svc.onRename?.eager?.(rProps);
      return { prevName };
    },
    mutationFn: useCallback<MutationFunction<void, { key: string; name: string }>>(
      async ({ key, name }: { key: string; name: string }, ...rest) => {
        const rProps = getRenameProps(key, name);
        const svc = services[rProps.id.type];
        if (svc.allowRename == null || svc.onRename == null) return;
        await svc?.onRename?.execute?.(getRenameProps(key, name), ...rest);
      },
      [services],
    ),
    onError: (error, { key, name }, ctx) => {
      if (ctx == null) return;
      const { prevName } = ctx;
      const rProps = getRenameProps(key, name);
      const svc = services[rProps.id.type];
      setNodes([
        ...Core.updateNode({
          tree: nodesRef.current,
          key,
          updater: (node) => ({ ...node, name: prevName }),
        }),
      ]);
      handleError(error, `Failed to rename ${prevName} to ${name}`);
      svc.onRename?.rollback?.(rProps, prevName);
    },
  });

  const handleRename = useCallback(
    (key: string, name: string) => rename.mutate({ key, name }),
    [rename],
  );

  const handleDoubleClick: Core.TreeProps["onDoubleClick"] = useCallback(
    (key: string) => {
      if (client == null) throw NULL_CLIENT_ERROR;
      const { type } = new ontology.ID(key);
      services[type].onSelect?.({
        client,
        store,
        services,
        placeLayout,
        handleError,
        removeLayout,
        addStatus,
        selection: resourcesRef.current.filter(({ id }) => id.toString() === key),
      });
    },
    [client, store, services, placeLayout, handleError, removeLayout, addStatus],
  );

  const handleContextMenu = useCallback(
    ({ keys }: Menu.ContextMenuMenuProps) => {
      if (keys.length === 0 || client == null) return <Layout.DefaultContextMenu />;
      const rightClickedButNotSelected = keys.find(
        (v) => !treeProps.selected.includes(v),
      );
      // In the case where we right clicked the menu, but it's not in the current
      // selection, we only display a context menu for that item.
      if (rightClickedButNotSelected != null) keys = [rightClickedButNotSelected];
      // Because we're using a virtualized tree, the keys from the context menu
      // might not actually be accurate (because we're missing DOM elements), so instead
      // we pull directly from the list selected state.
      else keys = selectedRef.current;
      const resources = resourcesRef.current;
      const nodeSnapshot = nodesRef.current;

      const selectedNodes = Core.findNodes({ tree: nodeSnapshot, keys });
      const selectedResources = resources.filter(({ key }) => keys.includes(key));

      // TODO: we might be selecting two nodes that are not ascendants or
      // descendants of the other ones. We need to change this function to
      // implement recursion.
      const parent = Core.findNodeParent({
        tree: nodeSnapshot,
        // We want to find the parent of the node with the lowest depth, since we
        // might be selecting nodes AND their children.
        key: selectedNodes.sort((a, b) => a.depth - b.depth)[0].key,
      });

      const parentID = parent == null ? root : new ontology.ID(parent.key);

      const firstID = new ontology.ID(keys[0]);

      const props: TreeContextMenuProps = {
        client,
        store,
        services,
        placeLayout,
        removeLayout,
        handleError,
        addStatus,
        selection: {
          rootID: root,
          parentID,
          nodes: selectedNodes,
          resources: selectedResources,
        },
        state: {
          nodes: nodeSnapshot,
          resources,
          setNodes,
          setSelection: setSelected,
          setResources,
          expand: treeProps.expand,
          contract: treeProps.contract,
          setLoading,
        },
      };

      const allSameType = selectedResources.every(({ id }) => id.type === firstID.type);
      if (!allSameType) return <MultipleSelectionContextMenu {...props} />;

      const M = services[firstID.type].TreeContextMenu;
      return M == null ? <Layout.DefaultContextMenu /> : <M {...props} />;
    },
    [
      client,
      setNodes,
      setResources,
      services,
      placeLayout,
      removeLayout,
      resourcesRef,
      nodesRef,
      treeProps.selected,
      setSelected,
    ],
  );

  const item = useCallback(
    (props: Core.ItemProps) => (
      <AdapterItem {...props} services={services} key={props.entry.path} />
    ),
    [services],
  );

  return (
    <>
      <Menu.ContextMenu menu={handleContextMenu} {...menuProps} />
      <Core.Tree
        onRename={handleRename}
        onDrop={handleDrop}
        onDoubleClick={handleDoubleClick}
        showRules
        loading={loading}
        virtual={false}
        onContextMenu={menuProps.open}
        className={menuProps.className}
        {...treeProps}
      >
        {item}
      </Core.Tree>
    </>
  );
};

interface AdapterItemProps extends Core.ItemProps {
  loading: boolean;
  services: Services;
}

const AdapterItem = memo<AdapterItemProps>(
  ({ loading, services, ...rest }): ReactElement => {
    const { type } = new ontology.ID(rest.entry.key);
    const Item = useMemo(() => services[type].Item ?? Core.DefaultItem, [type]);
    return <Item loading={loading} {...rest} />;
  },
);
AdapterItem.displayName = "AdapterItem";

export interface TreeProps {
  root?: ontology.ID | null;
}

export const Tree = ({ root }: TreeProps): ReactElement | null => {
  if (root == null) return null;
  return <Internal root={root} />;
};
