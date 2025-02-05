// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, ontology, type Synnax as Client } from "@synnaxlabs/client";
import {
  Haul,
  Menu,
  type state,
  Status,
  Synnax,
  useAsyncEffect,
  useCombinedStateAndRef,
  useStateRef as useRefAsState,
} from "@synnaxlabs/pluto";
import { Tree as Core } from "@synnaxlabs/pluto/tree";
import { deep, unique } from "@synnaxlabs/x";
import { type MutationFunction, useMutation } from "@tanstack/react-query";
import { Mutex } from "async-mutex";
import {
  isValidElement,
  memo,
  type ReactElement,
  useCallback,
  useMemo,
  useState,
} from "react";
import { useStore } from "react-redux";

import { Layout } from "@/layout";
import { MultipleSelectionContextMenu } from "@/ontology/ContextMenu";
import {
  type BaseProps,
  type HandleTreeRenameProps,
  type Services,
  type TreeContextMenuProps,
} from "@/ontology/service";
import { useServices } from "@/ontology/ServicesProvider";
import { type RootAction, type RootState } from "@/store";

export const toTreeNodes = (
  services: Services,
  resources: ontology.Resource[],
): Core.Node[] => resources.map((res) => toTreeNode(services, res));

export const toTreeNode = (
  services: Services,
  resource: ontology.Resource,
): Core.Node => {
  const { id, name } = resource;
  const { icon, hasChildren, haulItems } = services[id.type];
  return {
    key: id.toString(),
    name,
    icon: isValidElement(icon) ? icon : icon(resource),
    hasChildren,
    haulItems: haulItems(resource),
    allowRename: services[id.type].allowRename(resource),
  };
};

const updateResources = (
  p: ontology.Resource[],
  additions: ontology.Resource[] = [],
  removals: ontology.ID[] = [],
): ontology.Resource[] => {
  // If multiple additions have the same key, remove duplicates
  const uniqueAdditions = unique.by(
    additions,
    (resource) => resource.id.toString(),
    false,
  );
  const addedIds = uniqueAdditions.map(({ id }) => id.toString());
  const removedIds = removals.map((id) => id.toString());
  return [
    ...p.filter(({ id }) => {
      const str = id.toString();
      return !removedIds.includes(str) && !addedIds.includes(str);
    }),
    ...uniqueAdditions,
  ];
};

const loadInitialTree = async (
  client: Client,
  services: Services,
  setNodes: state.Set<Core.Node[]>,
  setResources: state.Set<ontology.Resource[]>,
): Promise<void> => {
  const fetched = await client.ontology.retrieveChildren(ontology.RootID, {
    includeSchema: true,
  });
  setNodes(toTreeNodes(services, fetched));
  setResources((p) => updateResources(p, fetched));
};

const mu = new Mutex();

const handleResourcesChange = async (
  changes: ontology.ResourceChange[],
  services: Services,
  nodes: Core.Node[],
  setNodes: state.Set<Core.Node[]>,
  setResources: state.Set<ontology.Resource[]>,
  resources: ontology.Resource[],
): Promise<void> =>
  await mu.runExclusive(async () => {
    const removed = changes
      .filter(({ variant }) => variant === "delete")
      .map(({ key }) => key);
    const updated = changes
      .filter(({ variant, value }) => variant === "set" && value != null)
      .map(({ value }) => value) as ontology.Resource[];
    setResources(updateResources(resources, updated, removed));
    let nextTree = Core.removeNode({
      tree: nodes,
      keys: removed.map((id) => id.toString()),
    });
    let changed = false;
    nextTree = updated.reduce(
      (nextTree, node) =>
        Core.updateNode({
          tree: nextTree,
          key: node.id.toString(),
          updater: (n) => {
            const next = { ...n, ...toTreeNode(services, node) };
            if (!changed && !deep.equal(next, n)) changed = true;
            return next;
          },
          throwOnMissing: false,
        }),
      nextTree,
    );
    if (changed) setNodes([...nextTree]);
  });

const handleRelationshipsChange = async (
  client: Client,
  changes: ontology.RelationshipChange[],
  services: Services,
  nodes: Core.Node[],
  setNodes: state.Set<Core.Node[]>,
  setResources: state.Set<ontology.Resource[]>,
  resources: ontology.Resource[],
): Promise<void> =>
  await mu.runExclusive(async () => {
    // Remove any relationships that were deleted
    const removed = changes
      .filter(({ variant, key: { type } }) => variant === "delete" && type === "parent")
      .map(({ key: { to } }) => to.toString());
    let nextTree = Core.removeNode({ tree: nodes, keys: removed });

    const allSets = changes
      .filter(({ variant, key: { type } }) => variant === "set" && type === "parent")
      .map(({ key }) => key);

    // Find all the parent nodes in the current tree that are visible i.e. they
    // may need children added.
    const visibleSetNodes = Core.findNodes({
      tree: nextTree,
      keys: allSets.map(({ from }) => from.toString()),
    }).map(({ key }) => key.toString());

    // Get all the relationships that relate to those visibe nodes.
    const visibleSets = allSets.filter(({ from }) =>
      visibleSetNodes.includes(from.toString()),
    );

    // Retrieve the new resources for the nodes that need to be updated.
    const updatedResources = await client.ontology.retrieve(
      visibleSets.map(({ to }) => to),
    );

    // Update the resources in the tree.
    setResources(updateResources(resources, updatedResources));

    // Update the tree.
    nextTree = visibleSets.reduce(
      (nextTree, { from, to }) =>
        Core.setNode({
          tree: nextTree,
          destination: from.toString(),
          additions: toTreeNodes(
            services,
            updatedResources.filter(({ id }) => id.toString() === to.toString()),
          ),
        }),
      nextTree,
    );

    setNodes([...nextTree]);
  });

const sortFunc = (a: Core.Node, b: Core.Node) => {
  const aIsGroup = a.key.startsWith(group.ONTOLOGY_TYPE);
  const bIsGroup = b.key.startsWith(group.ONTOLOGY_TYPE);
  if (aIsGroup && !bIsGroup) return -1;
  if (!aIsGroup && bIsGroup) return 1;
  return Core.defaultSort(a, b);
};

export const Tree = (): ReactElement => {
  const client = Synnax.use();
  const services = useServices();
  const store = useStore<RootState, RootAction>();
  const placeLayout = Layout.usePlacer();
  const removeLayout = Layout.useRemover();
  const [loading, setLoading] = useState<string | false>(false);
  const [nodes, setNodes, nodesRef] = useCombinedStateAndRef<Core.Node[]>([]);
  const [resourcesRef, setResources] = useRefAsState<ontology.Resource[]>([]);
  const [selected, setSelected, selectedRef] = useCombinedStateAndRef<string[]>([]);
  const addStatus = Status.useAggregator();
  const handleException = Status.useExceptionHandler();
  const menuProps = Menu.useContextMenu();

  const baseProps: BaseProps = useMemo<BaseProps>(
    () => ({
      client: client as Client,
      store,
      placeLayout,
      removeLayout,
      services,
      addStatus,
      handleException,
    }),
    [client, store, placeLayout, removeLayout, services, addStatus, handleException],
  );

  // Processes incoming changes to the ontology from the cluster.
  useAsyncEffect(async () => {
    if (client == null) return;
    await loadInitialTree(client, services, setNodes, setResources);

    const ct = await client.ontology.openChangeTracker();

    ct.resources.onChange((changes) => {
      void handleResourcesChange(
        changes,
        services,
        nodesRef.current,
        setNodes,
        setResources,
        resourcesRef.current,
      );
    });

    ct.relationships.onChange((changes) => {
      void handleRelationshipsChange(
        client,
        changes,
        services,
        nodesRef.current,
        setNodes,
        setResources,
        resourcesRef.current,
      );
    });

    return () => {
      void ct.close();
    };
  }, [client]);

  const handleExpand = useCallback(
    ({ action, clicked }: Core.HandleExpandProps): void => {
      if (action !== "expand") return;
      void (async () => {
        if (client == null) return;
        const id = new ontology.ID(clicked);
        try {
          setLoading(clicked);
          const resources = await client.ontology.retrieveChildren(id, {
            includeSchema: false,
          });
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
      })();
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
      handleException(error, "Failed to move resources");
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
      // Find the parent where the node is being dropped.
      const parent = Core.findNodeParent({
        tree: nodesSnapshot,
        key: firstNodeOfMinDepth.key.toString(),
      });
      if (parent == null) return [];
      const moved = dropped.filter(({ data }) => data?.depth === minDepth);
      const keys = moved.map(({ key }) => key as string);
      const sourceID = new ontology.ID(parent.key);
      treeProps.contract(...keys);
      dropMutation.mutate({
        source: sourceID,
        ids: keys.map((key) => new ontology.ID(key)),
        destination,
      });
      return moved;
    },
    [client, treeProps.contract],
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
      async ({ key, name }: { key: string; name: string }, ...props) => {
        const rProps = getRenameProps(key, name);
        const svc = services[rProps.id.type];
        if (svc.allowRename == null || svc.onRename == null) return;
        await svc?.onRename?.execute?.(getRenameProps(key, name), ...props);
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
      handleException(error, `Failed to rename ${prevName} to ${name}`);
      svc.onRename?.rollback?.(rProps, prevName);
    },
  });
  const handleRename = useCallback(
    (key: string, name: string) => rename.mutate({ key, name }),
    [rename],
  );

  const handleDoubleClick: Core.TreeProps["onDoubleClick"] = useCallback(
    (key: string) => {
      const id = new ontology.ID(key);
      const svc = services[id.type];
      if (client == null) return;
      void svc.onSelect?.({
        client,
        store,
        services,
        placeLayout,
        handleException,
        removeLayout,
        addStatus,
        selection: resourcesRef.current.filter(({ id }) => id.toString() === key),
      });
    },
    [client, store, placeLayout, removeLayout, resourcesRef],
  );

  const handleContextMenu = useCallback(
    ({ keys }: Menu.ContextMenuMenuProps): ReactElement | null => {
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

      const firstID = new ontology.ID(keys[0]);

      const props: TreeContextMenuProps = {
        client,
        store,
        services,
        placeLayout,
        removeLayout,
        handleException,
        addStatus,
        selection: { parent, nodes: selectedNodes, resources: selectedResources },
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
    (props: Core.ItemProps): ReactElement => (
      <AdapterItem {...props} key={props.entry.path} services={services} />
    ),
    [services],
  );

  return (
    <Menu.ContextMenu
      style={{ height: "calc(100% - 32px)" }}
      menu={handleContextMenu}
      {...menuProps}
    >
      <Core.Tree
        onRename={handleRename}
        onDrop={handleDrop}
        onDoubleClick={handleDoubleClick}
        showRules
        loading={loading}
        virtual={false}
        {...treeProps}
      >
        {item}
      </Core.Tree>
    </Menu.ContextMenu>
  );
};

interface AdapterItemProps extends Core.ItemProps {
  loading: boolean;
  services: Services;
}

const AdapterItem = memo<AdapterItemProps>(
  ({ loading, services, ...props }): ReactElement => {
    const id = new ontology.ID(props.entry.key);
    const Item = useMemo(() => services[id.type]?.Item ?? Core.DefaultItem, [id.type]);
    return <Item loading={loading} {...props} />;
  },
);
AdapterItem.displayName = "AdapterItem";
