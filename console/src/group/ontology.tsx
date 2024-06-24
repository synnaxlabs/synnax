// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Menu as PMenu } from "@synnaxlabs/pluto";
import { Tree } from "@synnaxlabs/pluto/tree";
import { type ReactElement } from "react";
import { v4 as uuid } from "uuid";

import { Menu } from "@/components/menu";
import { Ontology } from "@/ontology";

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { nodes, parent, resources },
  } = props;
  const onSelect = (key: string): void => {
    switch (key) {
      case "ungroup":
        void ungroupSelection(props);
        return;
      case "rename":
        Tree.startRenaming(nodes[0].key);
        return;
      case "group":
        void newGroup(props);
        return;
    }
  };

  const isDelete = nodes.every((n) => n.children == null || n.children.length === 0);
  const ungroupIcon = isDelete ? <Icon.Delete /> : <Icon.Group />;
  const singleResource = resources.length === 1;

  return (
    <PMenu.Menu onChange={onSelect} level="small" iconSpacing="small">
      <PMenu.Item itemKey="group" startIcon={<Icon.Group />}>
        New Group
      </PMenu.Item>
      {parent != null && (
        <PMenu.Item itemKey="ungroup" startIcon={ungroupIcon}>
          {/* TODO: Maybe we shouldn't force them into keeping the ontology tree like this? */}
          {isDelete ? "Delete" : "Ungroup"}
        </PMenu.Item>
      )}
      {singleResource && <Ontology.RenameMenuItem />}
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

export const UngroupMenuItem = (): ReactElement => (
  <PMenu.Item itemKey="ungroup" startIcon={<Icon.Group />}>
    Ungroup
  </PMenu.Item>
);

export interface GroupMenuItemProps {
  selection: Ontology.TreeContextMenuProps["selection"];
}

export const GroupMenuItem = ({
  selection,
}: GroupMenuItemProps): ReactElement | null =>
  canGroupSelection(selection) ? (
    <PMenu.Item itemKey="group" startIcon={<Icon.Group />}>
      Group
    </PMenu.Item>
  ) : null;

const ungroupSelection = async ({
  client,
  selection,
  state,
}: Ontology.TreeContextMenuProps): Promise<void> => {
  if (selection.parent == null) return;

  // Sort the groups by depth that way deeper nested groups are ungrouped first.
  selection.resources.sort((a, b) => {
    const a_depth = selection.nodes.find((n) => n.key === a.id.toString())?.depth ?? 0;
    const b_depth = selection.nodes.find((n) => n.key === b.id.toString())?.depth ?? 0;
    return b_depth - a_depth;
  });
  for (const node of selection.resources) {
    const id = node.id;
    const children =
      Tree.findNode({ tree: state.nodes, key: id.toString() })?.children ?? [];
    const parentID = new ontology.ID(selection.parent.key);
    state.setLoading(id.toString());
    await client.ontology.moveChildren(
      id,
      parentID,
      ...children.map((c) => new ontology.ID(c.key)),
    );
    await client.ontology.groups.delete(id.key);
    state.setLoading(false);
    let nextNodes = Tree.moveNode({
      tree: state.nodes,
      destination: parentID.toString(),
      keys: children.map((c) => c.key),
    });
    nextNodes = Tree.removeNode({ tree: nextNodes, keys: id.toString() });
    state.setNodes([...nextNodes]);
  }
};

const NEW_GROUP_NAME = "New Group Name";

export const canGroupSelection = (
  selection: Ontology.TreeContextMenuProps["selection"],
): boolean => getAllNodesOfMinDepth(selection.nodes).length > 1;

const getAllNodesOfMinDepth = (
  nodes: Tree.NodeWithPosition[],
): Tree.NodeWithPosition[] => {
  if (nodes.length === 0) return [];
  const depths = nodes.map(({ depth }) => depth).sort((a, b) => a - b);
  const minDepth = depths[0];
  return nodes.filter(({ depth }) => depth === minDepth);
};

export const newGroup = async ({
  client,
  state,
  services,
  selection: { resources },
}: Ontology.TreeContextMenuProps): Promise<void> => {
  if (resources.length === 0) return;
  const resource = resources[resources.length - 1];
  const otgID = new ontology.ID({ type: "group", key: uuid() });
  const res: ontology.Resource = {
    key: otgID.toString(),
    id: otgID,
    name: "",
  };
  state.expand(resource.id.toString());
  const newGroupNode = Ontology.toTreeNode(services, res);
  const nextNodes = Tree.setNode({
    tree: state.nodes,
    destination: resource.id.toString(),
    additions: newGroupNode,
  });
  state.setNodes([...nextNodes]);
  setTimeout(() => {
    Tree.startRenaming(res.id.toString(), async (name) => {
      if (name.length === 0) {
        // remove the node from the tree
        state.setNodes([
          ...Tree.removeNode({ tree: nextNodes, keys: res.id.toString() }),
        ]);
        return;
      }
      state.setLoading(otgID.toString());
      await client.ontology.groups.create(resource.id, name, otgID.key);
      state.setLoading(false);
    });
  }, 20);
};

export const fromSelection = async ({
  client,
  selection,
  services,
  state,
}: Ontology.TreeContextMenuProps): Promise<void> => {
  if (selection.parent == null) return;
  const nodesOfMinDepth = getAllNodesOfMinDepth(selection.nodes);
  const nodesOfMinDepthKeys = nodesOfMinDepth.map(({ key }) => key);
  const resourcesToGroup = selection.resources
    .filter(({ id }) => nodesOfMinDepthKeys.includes(id.toString()))
    .map(({ id }) => id);
  const parentID = new ontology.ID(selection.parent.key);
  const g = await client.ontology.groups.create(parentID, NEW_GROUP_NAME);
  const otgID = new ontology.ID({ type: "group", key: g.key.toString() });
  const res = await client.ontology.retrieve(otgID);
  await client.ontology.moveChildren(parentID, res.id, ...resourcesToGroup);
  const newGroupNode = Ontology.toTreeNode(services, res);
  let nextNodes = Tree.setNode({
    tree: state.nodes,
    destination: selection.parent.key,
    additions: newGroupNode,
  });
  nextNodes = Tree.moveNode({
    tree: state.nodes,
    destination: res.id.toString(),
    keys: resourcesToGroup.map((id) => id.toString()),
  });
  state.setNodes([...nextNodes]);
  state.setResources([...state.resources, res]);
  state.setSelection([res.id.toString()]);
  setTimeout(() => {
    Tree.startRenaming(res.id.toString());
  }, 20);
};

const handleRename: Ontology.HandleTreeRename = ({
  client,
  id,
  name,
  state: { nodes, setNodes },
}) => {
  void (async () => {
    if (client == null || id.type !== "group") return;
    await client.ontology.groups.rename(id.key, name);
    const next = Tree.updateNode({
      tree: nodes,
      key: id.toString(),
      updater: (node) => ({
        ...node,
        name,
      }),
    });
    setNodes([...next]);
  })();
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: "group",
  icon: <Icon.Group />,
  hasChildren: true,
  onRename: handleRename,
  canDrop: () => true,
  onSelect: () => {},
  haulItems: () => [],
  allowRename: () => true,
  onMosaicDrop: () => {},
  TreeContextMenu,
};
