// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { ontology, type ChannelKey, type Synnax } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Haul, Menu, Tree } from "@synnaxlabs/pluto";
import { MenuProps } from "@synnaxlabs/pluto/dist/menu/Menu.js";

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
  },
  channel: {
    type: "channel",
    icon: <Icon.Channel />,
    hasChildren: false,
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
    contextMenu: (ctx) => {
      return (
        <Menu.Menu>
          <Menu.Item
            level="small"
            itemKey="group"
            startIcon={<Icon.Group />}
            iconSpacing="small"
            onClick={() => {}}
          >
            Group Selection
          </Menu.Item>
          <Menu.Item
            itemKey="rename"
            level="small"
            startIcon={<Icon.Edit />}
            iconSpacing="small"
          >
            Rename
          </Menu.Item>
        </Menu.Menu>
      );
    },
  },
  group: {
    type: "group",
    hasChildren: true,
    icon: <Icon.Group />,
    canDrop: () => true,
    onSelect: () => {},
    contextMenu: () => <></>,
    haulItems: () => [],
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
