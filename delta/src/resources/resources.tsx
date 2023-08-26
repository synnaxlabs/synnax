// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import {
  ontology,
  type ChannelKey,
  type Synnax,
  UnexpectedError,
} from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Haul, Menu, Tree, Text } from "@synnaxlabs/pluto";
import { rename } from "@synnaxlabs/pluto/dist/tabs/Tabs.js";

import { LayoutPlacer, selectActiveMosaicLayout } from "@/layout";
import {
  ZERO_CHANNELS_STATE,
  addLinePlotYChannel,
  createLinePlot,
} from "@/line/store/slice";
import { RootStore } from "@/store";
import { addRange } from "@/workspace";

export interface ResourceSelectionContext {
  client: Synnax;
  store: RootStore;
  placeLayout: LayoutPlacer;
  selection: {
    parent: Tree.Node;
    resources: ontology.Resource[];
    nodes: Tree.Node[];
  };
  state: {
    resources: ontology.Resource[];
    nodes: Tree.Node[];
    setNodes: (nodes: Tree.Node[]) => void;
  };
}

export interface ResourceType {
  type: ontology.ResourceType;
  icon: ReactElement;
  hasChildren: boolean;
  allowRename: (res: ontology.Resource) => boolean;
  onSelect: (ctx: ResourceSelectionContext) => void;
  canDrop: (hauled: Haul.Item[]) => boolean;
  onDrop: (ctx: ResourceSelectionContext, hauled: Haul.Item[]) => void;
  contextMenu: (ctx: ResourceSelectionContext) => ReactElement;
  haulItems: (resource: ontology.Resource) => Haul.Item[];
}

export const convertOntologyResources = (
  resources: ontology.Resource[]
): Tree.Node[] => {
  return resources.map((res) => {
    const { id, name } = res;
    const { icon, hasChildren, haulItems } = resourceTypes[id.type];
    return {
      key: id.toString(),
      name,
      icon,
      hasChildren,
      children: [],
      haulItems: haulItems(res),
      allowRename: resourceTypes[id.type].allowRename(res),
    };
  });
};

export const resourceTypes: Record<string, ResourceType> = {
  builtin: {
    type: "builtin",
    icon: <Icon.Cluster />,
    hasChildren: true,
    canDrop: () => false,
    onDrop: () => {},
    contextMenu: () => <></>,
    onSelect: () => {},
    haulItems: () => [],
    allowRename: () => false,
  },
  cluster: {
    type: "cluster",
    icon: <Icon.Cluster />,
    hasChildren: true,
    canDrop: () => false,
    onDrop: () => {},
    contextMenu: () => <></>,
    onSelect: () => {},
    haulItems: () => [],
    allowRename: () => false,
  },
  node: {
    type: "node",
    icon: <Icon.Node />,
    hasChildren: true,
    canDrop: () => false,
    onDrop: () => {},
    contextMenu: () => <></>,
    onSelect: () => {},
    haulItems: () => [],
    allowRename: () => false,
  },
  channel: {
    type: "channel",
    icon: <Icon.Channel />,
    hasChildren: false,
    allowRename: () => true,
    canDrop: () => false,
    onDrop: () => {},
    onSelect: (ctx) => {
      const s = ctx.store.getState();
      const layout = selectActiveMosaicLayout(s);
      if (layout == null) {
        ctx.placeLayout(
          createLinePlot({
            channels: {
              ...ZERO_CHANNELS_STATE,
              y1: [ctx.selected.data.key as ChannelKey],
            },
          })
        );
      }
      switch (layout?.type) {
        case "line":
          ctx.store.dispatch(
            addLinePlotYChannel({
              key: layout?.key,
              axisKey: "y1",
              channels: [ctx.selected.data.key as ChannelKey],
            })
          );
      }
    },
    haulItems: (res) => {
      return [
        {
          type: "channel",
          key: Number(res.id.key),
        },
      ];
    },
    contextMenu: (ctx) => <></>,
  },
  group: {
    type: "group",
    hasChildren: true,
    icon: <Icon.Group />,
    canDrop: () => true,
    onSelect: () => {},
    contextMenu: (ctx) => {
      const onSelect = (key: string): void => {
        switch (key) {
          case "ungroup":
            void ungroupSelection(ctx);
            return;
          case "rename":
            startRenaming(ctx);
        }
      };

      return (
        <Menu.Menu onChange={onSelect} level="small" iconSpacing="small">
          <Menu.Item itemKey="ungroup" startIcon={<Icon.Group />}>
            Ungroup
          </Menu.Item>
          <Menu.Item itemKey="rename" startIcon={<Icon.Edit />}>
            Rename
          </Menu.Item>
        </Menu.Menu>
      );
    },
    haulItems: () => [],
    onDrop: () => {},
    allowRename: () => true,
  },
  range: {
    type: "range",
    hasChildren: true,
    icon: <Icon.Range />,
    canDrop: () => true,
    onDrop: () => {},
    onSelect: (ctx) => {
      ctx.store.dispatch(
        addRange({
          name: ctx.selected.data.name,
          type: "static",
          key: ctx.selected.data.key,
          timeRange: ctx.selected.data.timeRange,
        })
      );
    },
    contextMenu: () => <></>,
    haulItems: () => [],
  },
};

const GroupSelectionMenuItem = (): ReactElement => (
  <Menu.Item itemKey="group" startIcon={<Icon.Group />}>
    Group Selection
  </Menu.Item>
);

const NEW_GROUP_NAME = "New Group";

const startRenaming = ({ selection, state }: ResourceSelectionContext): void =>
  Text.edit(`text-${selection.nodes[0].key}`);

const ungroupSelection = async ({
  client,
  selection,
  state,
}: ResourceSelectionContext): Promise<void> => {
  if (selection.resources.length !== 1)
    throw new UnexpectedError("[ungroupSelection] - expected exactly one resource");

  const id = selection.resources[0].id;
  const children = Tree.findNode(state.nodes, id.toString())?.children ?? [];
  const parentID = new ontology.ID(selection.parent.key);
  await client.ontology.moveChildren(
    id,
    parentID,
    ...children.map((c) => new ontology.ID(c.key))
  );
  await client.ontology.groups.delete(id.key);
  let nextNodes = Tree.moveNode(
    state.nodes,
    parentID.toString(),
    ...children.map((c) => c.key)
  );
  nextNodes = Tree.removeNode(nextNodes, id.toString());
  state.setNodes([...nextNodes]);
};

const groupSelection = async ({
  client,
  selection,
  state,
}: ResourceSelectionContext): Promise<void> => {
  const parentID = new ontology.ID(selection.parent.key);
  const g = await client.ontology.groups.create(parentID, NEW_GROUP_NAME);
  const otgID = new ontology.ID({ type: "group", key: g.key.toString() });
  const res = await client.ontology.retrieve(otgID);
  const selectionIDs = selection.resources.map(({ id }) => id);
  await client.ontology.moveChildren(parentID, res.id, ...selectionIDs);
  let nextNodes = Tree.addNode(
    state.nodes,
    selection.parent.key,
    ...convertOntologyResources([res])
  );
  nextNodes = Tree.moveNode(
    state.nodes,
    res.id.toString(),
    ...selectionIDs.map((id) => id.toString())
  );
  state.setNodes([...nextNodes]);
};

export const MultipleSelectionContextMenu = (
  ctx: ResourceSelectionContext
): ReactElement => {
  const handleSelect: MenuProps["onChange"] = (key) => {
    switch (key) {
      case "group":
        void groupSelection(ctx);
    }
  };

  return (
    <Menu.Menu onChange={handleSelect} level="small" iconSpacing="small">
      <GroupSelectionMenuItem />
    </Menu.Menu>
  );
};
