// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useRef, useState } from "react";

import { ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Tree, Align, Synnax, useAsyncEffect, Menu, Haul } from "@synnaxlabs/pluto";
import { useStore } from "react-redux";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { Layout } from "@/layout";
import {
  MultipleSelectionContextMenu,
  ResourceSelectionContext,
  convertOntologyResources,
  types,
} from "@/resources/resources";
import { Action, RootState } from "@/store";

const ResourcesTree = (): ReactElement => {
  const client = Synnax.use();
  const [nodes, setNodes] = useState<Tree.Node[]>([]);
  const store = useStore<RootState, Action>();
  const placer = Layout.usePlacer();
  const resourcesRef = useRef<ontology.Resource[]>([]);

  useAsyncEffect(async () => {
    if (client == null) return;
    const resources = await client.ontology.retrieveChildren(ontology.Root, true, true);
    setNodes(convertOntologyResources(resources));
    const keys = resources.map(({ id }) => id.toString());
    resourcesRef.current = [
      ...resourcesRef.current.filter(({ id }) => !keys.includes(id.toString())),
      ...resources,
    ];
  }, [client]);

  const handleExpand = ({ action, clicked }: Tree.HandleExpandProps): void => {
    if (action !== "expand") return;
    void (async () => {
      if (client == null) return;
      let nextTree: Tree.Node[] = [...nodes];
      const resources = await client?.ontology.retrieveChildren(
        new ontology.ID(clicked),
        true,
        true
      );
      const converted = convertOntologyResources(resources);
      nextTree = Tree.addNode(nextTree, clicked, ...converted);
      const keys = resources.map(({ id }) => id.toString());
      resourcesRef.current = [
        ...resourcesRef.current.filter(({ id }) => !keys.includes(id.toString())),
        ...resources,
      ];
      setNodes(nextTree);
    })();
  };

  const props = Tree.use({ onExpand: handleExpand });

  const menuProps = Menu.useContextMenu();

  const handleDrop: Tree.TreeProps["onDrop"] = (
    key,
    { source, items }
  ): Haul.Item[] => {
    const dropped = Haul.filterByType(Tree.HAUL_TYPE, items);
    if (dropped.length === 0 || source.type !== "Tree.Item" || client == null)
      return [];
    void (async () => {
      const otgID = new ontology.ID(key);
      const parent = Tree.findNodeParent(nodes, source.key as string);
      if (parent == null) return;
      await client.ontology.moveChildren(
        new ontology.ID(parent.key),
        otgID,
        ...dropped.map(({ key }) => new ontology.ID(key as string))
      );
      const next = Tree.moveNode(
        nodes,
        key,
        ...dropped.map(({ key }) => key as string)
      );
      setNodes([...next]);
    })();
    return dropped;
  };

  const handleRename = (key: string, name: string): void => {
    void (async () => {
      const otgID = new ontology.ID(key);
      if (client == null || otgID.type !== "group") return;
      await client.ontology.groups.rename(otgID.key, name);
      const next = Tree.updateNode(nodes, key, (node) => ({
        ...node,
        name,
      }));
      setNodes([...next]);
    })();
  };

  return (
    <Align.Space empty style={{ height: "100%", position: "relative" }}>
      <ToolbarHeader>
        <ToolbarTitle icon={<Icon.Resources />}>Resources</ToolbarTitle>
      </ToolbarHeader>
      <Menu.ContextMenu
        menu={({ keys }) => {
          if (keys.length === 0 || client == null) return <></>;
          const selectedNodes = Tree.findNodes(nodes, keys);
          const selectedResources = resourcesRef.current.filter(({ id }) =>
            keys.includes(id.toString())
          );
          const parent = Tree.findNodeParent(nodes, keys[0]);
          if (parent == null) return <></>;

          const ctx: ResourceSelectionContext = {
            client,
            store,
            placeLayout: placer,
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

          if (selectedNodes.length > 1)
            return <MultipleSelectionContextMenu {...ctx} />;
          return types[new ontology.ID(keys[0]).type].contextMenu(ctx);
        }}
        {...menuProps}
      >
        <Tree.Tree
          onDrop={handleDrop}
          nodes={nodes}
          onRename={handleRename}
          {...props}
          style={{ height: "calc(100% - 100px)" }}
        />
      </Menu.ContextMenu>
    </Align.Space>
  );
};

export const Toolbar: Layout.NavDrawerItem = {
  key: "resources",
  icon: <Icon.Resources />,
  content: <ResourcesTree />,
  tooltip: "Resources",
  initialSize: 350,
  minSize: 250,
  maxSize: 650,
};
