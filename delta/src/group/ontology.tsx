// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError, ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Menu, Tree } from "@synnaxlabs/pluto";

import { Ontology } from "@/ontology";

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { resources },
  } = props;
  const onSelect = (key: string): void => {
    switch (key) {
      case "ungroup":
        void ungroupSelection(props);
        return;
      case "rename":
        Tree.startRenaming(resources[0].key);
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
};

const ungroupSelection = async ({
  client,
  selection,
  state,
}: Ontology.TreeContextMenuProps): Promise<void> => {
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

export const fromSelection = async ({
  client,
  selection,
  state,
}: Ontology.TreeContextMenuProps): Promise<void> => {
  const parentID = new ontology.ID(selection.parent.key);
  const g = await client.ontology.groups.create(parentID, NEW_GROUP_NAME);
  const otgID = new ontology.ID({ type: "group", key: g.key.toString() });
  const res = await client.ontology.retrieve(otgID);
  const selectionIDs = selection.resources.map(({ id }) => id);
  await client.ontology.moveChildren(parentID, res.id, ...selectionIDs);
  let nextNodes = Tree.addNode(
    state.nodes,
    selection.parent.key,
    ...Ontology.toTreeNodes([res])
  );
  nextNodes = Tree.moveNode(
    state.nodes,
    res.id.toString(),
    ...selectionIDs.map((id) => id.toString())
  );
  state.setNodes([...nextNodes]);
};

const handleRename: Ontology.HnadleTreeRename = ({
  client,
  id,
  name,
  state: { nodes, setNodes },
}) => {
  void (async () => {
    if (client == null || id.type !== "group") return;
    await client.ontology.groups.rename(id.key, name);
    const next = Tree.updateNode(nodes, id.key, (node) => ({
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
  allowRename: () => false,
  onMosaicDrop: () => {},
  TreeContextMenu,
};
