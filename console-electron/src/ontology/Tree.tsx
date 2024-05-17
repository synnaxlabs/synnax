// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useCallback, useState, memo, useMemo } from "react";

import { ontology, type Synnax as Client } from "@synnaxlabs/client";
import {
  Menu,
  Synnax,
  useAsyncEffect,
  Haul,
  useStateRef as useRefAsState,
  useCombinedStateAndRef,
  type state,
} from "@synnaxlabs/pluto";
import { Tree as Core } from "@synnaxlabs/pluto/tree";
import { deep } from "@synnaxlabs/x";
import { useStore } from "react-redux";

import { Layout } from "@/layout";
import { MultipleSelectionContextMenu } from "@/ontology/ContextMenu";
import { type Services, type TreeContextMenuProps } from "@/ontology/service";
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
    icon,
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
  const newIds = additions.map(({ id }) => id.toString());
  const removalIds = removals.map((id) => id.toString());
  return [
    ...p.filter(({ id }) => {
      const str = id.toString();
      return !removalIds.includes(str) && !newIds.includes(str);
    }),
    ...additions,
  ];
};

const loadInitialTree = async (
  client: Client,
  services: Services,
  setNodes: state.Set<Core.Node[]>,
  setResources: state.Set<ontology.Resource[]>,
): Promise<void> => {
  const fetched = await client.ontology.retrieveChildren(ontology.Root, true, true);
  setNodes(toTreeNodes(services, fetched));
  setResources((p) => updateResources(p, fetched));
};

const handleResourcesChange = async (
  changes: ontology.ResourceChange[],
  services: Services,
  nodes: Core.Node[],
  setNodes: state.Set<Core.Node[]>,
  setResources: state.Set<ontology.Resource[]>,
  resources: ontology.Resource[],
): Promise<void> => {
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
};

const handleRelationshipsChange = async (
  client: Client,
  changes: ontology.RelationshipChange[],
  services: Services,
  nodes: Core.Node[],
  setNodes: state.Set<Core.Node[]>,
  setResources: state.Set<ontology.Resource[]>,
  resources: ontology.Resource[],
): Promise<void> => {
  // Remove any relationships that were deleted
  const removed = changes
    .filter(({ variant }) => variant === "delete")
    .map(({ key: { to } }) => to.toString());
  let nextTree = Core.removeNode({ tree: nodes, keys: removed });

  const allSets = changes
    .filter(({ variant }) => variant === "set")
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
  const [selected, setSelected] = useState<string[]>([]);

  const menuProps = Menu.useContextMenu();

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

  const handleDrop: Core.TreeProps["onDrop"] = useCallback(
    (key: string, { source, items }: Haul.OnDropProps): Haul.Item[] => {
      const nodesSnapshot = nodesRef.current;
      const dropped = Haul.filterByType(Core.HAUL_TYPE, items);
      const isValidDrop = dropped.length > 0 && source.type === "Tree.Item";
      if (!isValidDrop) return [];
      const otgID = new ontology.ID(key);
      const svc = services[otgID.type];
      if (!svc.canDrop({ source, items })) return [];
      // Find the parent where the node is being dropped.
      const parent = Core.findNodeParent({
        tree: nodesSnapshot,
        key: source.key.toString(),
      });
      if (parent == null) return [];
      void (async () => {
        if (client == null) return;
        // Move the children in the ontology.
        await client.ontology.moveChildren(
          new ontology.ID(parent.key),
          otgID,
          ...dropped.map(({ key }) => new ontology.ID(key as string)),
        );
        // Move the nodes in the tree.
        const next = Core.moveNode({
          tree: nodesSnapshot,
          destination: key,
          keys: dropped.map(({ key }) => key as string),
        });
        setNodes([...next]);
      })();
      return dropped;
    },
    [client],
  );

  const handleExpand = useCallback(
    ({ action, clicked }: Core.HandleExpandProps): void => {
      if (action !== "expand") return;
      void (async () => {
        if (client == null) return;
        const id = new ontology.ID(clicked);
        try {
          setLoading(clicked);
          const resources = await client.ontology.retrieveChildren(id, true, true);
          const converted = toTreeNodes(services, resources);
          const nextTree = Core.setNode({
            tree: nodesRef.current,
            destination: clicked,
            additions: converted,
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

  const handleRename: Core.TreeProps["onRename"] = useCallback(
    (key: string, name: string) => {
      const id = new ontology.ID(key);
      const svc = services[id.type];
      if (client == null) return;
      const nodesSnapshot = nodesRef.current;
      svc.onRename?.({
        id,
        services,
        client,
        store,
        placeLayout,
        removeLayout,
        name,
        state: {
          nodes: nodesSnapshot,
          resources: resourcesRef.current,
          setNodes,
          setResources,
        },
      });
    },
    [services, client, store, placeLayout, removeLayout, resourcesRef],
  );

  const handleDoubleClick: Core.TreeProps["onDoubleClick"] = useCallback(
    (key: string) => {
      const id = new ontology.ID(key);
      const svc = services[id.type];
      if (client == null) return;
      svc.onSelect({
        client,
        store,
        services,
        placeLayout,
        removeLayout,
        selection: resourcesRef.current.filter(({ id }) => id.toString() === key),
      });
    },
    [client, store, placeLayout, removeLayout, resourcesRef],
  );

  const treeProps = Core.use({
    onExpand: handleExpand,
    nodes,
    selected,
    onSelectedChange: setSelected,
  });

  const handleContextMenu = useCallback(
    ({ keys }: Menu.ContextMenuMenuProps): ReactElement | null => {
      if (keys.length === 0 || client == null) return null;
      const rightClickedButNotSelected = keys.find(
        (v) => !treeProps.selected.includes(v),
      );
      // In the case where we right clicked the menu, but it's not in the current
      // selection, we only display a context menu for that item.
      if (rightClickedButNotSelected != null) keys = [rightClickedButNotSelected];
      const resources = resourcesRef.current;
      const nodeSnapshot = nodesRef.current;

      const selectedNodes = Core.findNodes({ tree: nodeSnapshot, keys });
      const selectedResources = resources.filter(({ key }) => keys.includes(key));

      const parent = Core.findNodeParent({
        tree: nodeSnapshot,
        // We want to find the parent of the node with the lowest depth, since we
        // might be selecting nodes AND their children.
        key: selectedNodes.sort((a, b) => a.depth - b.depth)[0].key,
      });
      // No parent means no valid contex menu.
      if (parent == null) return null;

      const firstID = new ontology.ID(keys[0]);

      const props: TreeContextMenuProps = {
        client,
        store,
        services,
        placeLayout,
        removeLayout,
        selection: {
          parent,
          nodes: selectedNodes,
          resources: selectedResources,
        },
        state: {
          nodes: nodeSnapshot,
          resources,
          setNodes,
          setSelection: setSelected,
          setResources,
        },
      };

      const allSameType = selectedResources.every(({ id }) => id.type === firstID.type);
      if (!allSameType) return <MultipleSelectionContextMenu {...props} />;

      const M = services[firstID.type].TreeContextMenu;
      return M == null ? null : <M {...props} />;
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
      <AdapterItem
        loading={props.entry.key === loading}
        services={services}
        {...props}
      />
    ),
    [services, loading],
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
    return <Item key={props.entry.key} loading={loading} {...props} />;
  },
);
AdapterItem.displayName = "AdapterItem";
