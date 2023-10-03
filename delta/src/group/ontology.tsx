// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { UnexpectedError, ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Menu, Tree } from "@synnaxlabs/pluto";

import { Ontology } from "@/ontology";

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { resources, nodes },
  } = props;
  console.log(resources);
  const onSelect = (key: string): void => {
    switch (key) {
      case "ungroup":
        void ungroupSelection(props);
        return;
      case "rename":
        Tree.startRenaming(nodes[0].key);
    }
  };

  return (
    <Menu.Menu onChange={onSelect} level="small" iconSpacing="small">
      <Menu.Item itemKey="ungroup" startIcon={<Icon.Group />}>
        Ungroup
      </Menu.Item>
      <Ontology.RenameMenuItem />
    </Menu.Menu>
  );
};

export const UngroupMenuItem = (): ReactElement => (
  <Menu.Item itemKey="ungroup" startIcon={<Icon.Group />}>
    Ungroup
  </Menu.Item>
);

export interface GroupMenuItemProps {
  selection: Ontology.TreeContextMenuProps["selection"];
}

export const GroupMenuItem = ({ selection }: GroupMenuItemProps): ReactElement | null =>
  canGroupSelection(selection) ? (
    <Menu.Item itemKey="group" startIcon={<Icon.Group />}>
      Group
    </Menu.Item>
  ) : null;

const ungroupSelection = async ({
  client,
  selection,
  state,
}: Ontology.TreeContextMenuProps): Promise<void> => {
  console.log(selection);
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

const NEW_GROUP_NAME = "New Group";

export const canGroupSelection = (
  selection: Ontology.TreeContextMenuProps["selection"]
): boolean => getAllNodesOfMinDepth(selection.nodes).length > 1;

const getAllNodesOfMinDepth = (nodes: Tree.NodeWithDepth[]): Tree.NodeWithDepth[] => {
  if (nodes.length === 0) return [];
  const depths = nodes.map(({ depth }) => depth).sort((a, b) => a - b);
  const minDepth = depths[0];
  return nodes.filter(({ depth }) => depth === minDepth);
};

export const fromSelection = async ({
  client,
  selection,
  services,
  state,
}: Ontology.TreeContextMenuProps): Promise<void> => {
  const nodesOfMinDepth = getAllNodesOfMinDepth(selection.nodes).map(({ key }) => key);
  const resourcesToGroup = selection.resources
    .filter(({ id }) => nodesOfMinDepth.includes(id.toString()))
    .map(({ id }) => id);
  const parentID = new ontology.ID(selection.parent.key);
  const g = await client.ontology.groups.create(parentID, NEW_GROUP_NAME);
  const otgID = new ontology.ID({ type: "group", key: g.key.toString() });
  const res = await client.ontology.retrieve(otgID);
  await client.ontology.moveChildren(parentID, res.id, ...resourcesToGroup);
  let nextNodes = Tree.addNode(
    state.nodes,
    selection.parent.key,
    ...Ontology.toTreeNodes(services, [res])
  );
  nextNodes = Tree.moveNode(
    state.nodes,
    res.id.toString(),
    ...resourcesToGroup.map((id) => id.toString())
  );
  state.setNodes([...nextNodes]);
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
    const next = Tree.updateNode(nodes, id.toString(), (node) => ({
      ...node,
      name,
    }));
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
