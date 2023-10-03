// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState, type ReactElement, useRef, useCallback } from "react";

import { ontology } from "@synnaxlabs/client";
import { Menu, Tree as Core, Synnax, useAsyncEffect, Haul } from "@synnaxlabs/pluto";
import { useStore } from "react-redux";

import { Layout } from "@/layout";
import { MultipleSelectionContextMenu } from "@/ontology/ContextMenu";
import { type Services, type TreeContextMenuProps } from "@/ontology/service";
import { useServices } from "@/ontology/ServicesProvider";
import { type RootAction, type RootState } from "@/store";

export const toTreeNodes = (
  services: Services,
  resources: ontology.Resource[]
): Core.Node[] => {
  return resources.map((res) => {
    const { id, name } = res;
    const { icon, hasChildren, haulItems } = services[id.type];
    return {
      key: id.toString(),
      name,
      icon,
      hasChildren,
      children: [],
      haulItems: haulItems(res),
      allowRename: services[id.type].allowRename(res),
    };
  });
};

export const Tree = (): ReactElement => {
  const client = Synnax.use();
  const services = useServices();
  const [nodes, setNodes] = useState<Core.Node[]>([]);
  const store = useStore<RootState, RootAction>();
  const placeLayout = Layout.usePlacer();
  const removeLayout = Layout.useRemover();
  const resourcesRef = useRef<ontology.Resource[]>([]);
  const menuProps = Menu.useContextMenu();

  // Load in the initial tree.
  useAsyncEffect(async () => {
    if (client == null) return;
    const resources = await client.ontology.retrieveChildren(ontology.Root, true, true);
    setNodes(toTreeNodes(services, resources));
    const keys = resources.map(({ id }) => id.toString());
    resourcesRef.current = [
      ...resourcesRef.current.filter(({ id }) => !keys.includes(id.toString())),
      ...resources,
    ];
  }, [client]);

  const handleDrop: Core.TreeProps["onDrop"] = useCallback(
    (key: string, { source, items }: Haul.OnDropProps): Haul.Item[] => {
      const dropped = Haul.filterByType(Core.HAUL_TYPE, items);
      if (dropped.length === 0 || source.type !== "Tree.Item" || client == null)
        return [];
      void (async () => {
        const otgID = new ontology.ID(key);
        const parent = Core.findNodeParent(nodes, source.key as string);
        if (parent == null) return;
        await client.ontology.moveChildren(
          new ontology.ID(parent.key),
          otgID,
          ...dropped.map(({ key }) => new ontology.ID(key as string))
        );
        const next = Core.moveNode(
          nodes,
          key,
          ...dropped.map(({ key }) => key as string)
        );
        setNodes([...next]);
      })();
      return dropped;
    },
    [nodes, client]
  );

  const handleExpand = useCallback(
    ({ action, clicked }: Core.HandleExpandProps): void => {
      if (action !== "expand") return;
      void (async () => {
        if (client == null) return;
        let nextTree: Core.Node[] = [...nodes];
        const id = new ontology.ID(clicked);
        const resources = await client.ontology.retrieveChildren(id, true, true);
        const converted = toTreeNodes(services, resources);
        nextTree = Core.addNode(nextTree, clicked, ...converted);
        const keys = resources.map(({ id }) => id.toString());
        resourcesRef.current = [
          ...resourcesRef.current.filter(({ id }) => !keys.includes(id.toString())),
          ...resources,
        ];
        setNodes(nextTree);
      })();
    },
    [client, nodes, services]
  );

  const handleRename: Core.TreeProps["onRename"] = useCallback(
    (key: string, name: string) => {
      const id = new ontology.ID(key);
      const svc = services[id.type];
      if (svc.onRename == null || client == null) return;
      svc.onRename({
        id,
        services,
        client,
        store,
        placeLayout,
        removeLayout,
        name,
        state: {
          nodes,
          resources: resourcesRef.current,
          setNodes,
        },
      });
    },
    [services, client, store, placeLayout, removeLayout, resourcesRef, client]
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
    [client, store, placeLayout, removeLayout, resourcesRef, client]
  );

  const treeProps = Core.use({ onExpand: handleExpand });

  return (
    <Menu.ContextMenu
      style={{ height: "calc(100% - 32px)" }}
      menu={({ keys }) => {
        if (keys.length === 0 || client == null) return <></>;
        if (keys.some((v) => !treeProps.selected.includes(v))) keys = [keys[0]];
        const selectedNodes = Core.findNodes(nodes, keys);
        const selectedResources = resourcesRef.current.filter(({ id }) =>
          keys.includes(id.toString())
        );

        const parent = Core.findNodeParent(
          nodes,
          selectedNodes.sort((a, b) => a.depth - b.depth)[0].key
        );
        if (parent == null) return null;
        const otgID = new ontology.ID(keys[0]);

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
            nodes,
            resources: resourcesRef.current,
            setNodes,
          },
        };

        const allSameType = selectedResources.every(({ id }) => id.type === otgID.type);
        if (!allSameType) return <MultipleSelectionContextMenu {...props} />;

        const CtxMenu = services[otgID.type].TreeContextMenu;
        if (CtxMenu == null) return null;

        return <CtxMenu {...props} />;
      }}
      {...menuProps}
    >
      <Core.Tree
        onRename={handleRename}
        onDrop={handleDrop}
        nodes={nodes}
        onDoubleClick={handleDoubleClick}
        {...treeProps}
      />
    </Menu.ContextMenu>
  );
};
