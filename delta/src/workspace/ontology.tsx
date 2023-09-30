// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";
import { Menu, Tree } from "@synnaxlabs/pluto";

import { Group } from "@/group";
import { Layout } from "@/layout";
import { Ontology } from "@/ontology";
import { add, rename, setActive } from "@/workspace/slice";

import { selectActiveKey } from "./selectors";

const handleDelete = ({
  client,
  store,
  selection: { resources },
  state: { nodes, setNodes },
}: Ontology.TreeContextMenuProps): void => {
  void (async () => {
    const s = store.getState();
    const activeKey = selectActiveKey(s);
    const active = resources.find((r) => r.id.key === activeKey);
    if (active != null) {
      store.dispatch(setActive(null));
      store.dispatch(Layout.setWorkspace({ slice: Layout.ZERO_SLICE_STATE }));
    }
    await client.workspaces.delete(...resources.map((r) => r.id.key));
    const next = Tree.removeNode(nodes, ...resources.map((r) => r.id.toString()));
    setNodes([...next]);
  })();
};

const TreeContextMenu: Ontology.TreeContextMenu = (props): ReactElement => {
  const {
    selection: { resources },
  } = props;
  const handleSelect = (key: string): void => {
    switch (key) {
      case "delete":
        return handleDelete(props);
      case "rename":
        return Tree.startRenaming(resources[0].id.toString());
      case "group":
        void Group.fromSelection(props);
    }
  };

  return (
    <Menu.Menu onChange={handleSelect} level="small" iconSpacing="small">
      <Menu.Item itemKey="delete" startIcon={<Icon.Delete />}>
        Delete
      </Menu.Item>
      <Ontology.RenameMenuItem />
      <Group.GroupMenuItem selection={props.selection} />
    </Menu.Menu>
  );
};

const handleSelect: Ontology.HandleSelect = ({ selection, client, store }) => {
  void (async () => {
    const ws = await client.workspaces.retrieve(selection[0].id.key);
    store.dispatch(add({ workspaces: [ws] }));
    store.dispatch(
      Layout.setWorkspace({
        slice: ws.layout as unknown as Layout.SliceState,
        keepNav: false,
      })
    );
  })();
};

const handleRename: Ontology.HnadleTreeRename = ({
  client,
  id,
  name,
  store,
  state: { nodes, setNodes },
}) => {
  void client.workspaces.rename(id.key, name);
  store.dispatch(rename({ key: id.key, name }));

  const next = Tree.updateNode(nodes, id.toString(), (node) => ({ ...node, name }));
  setNodes([...next]);
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: "workspace",
  icon: <Icon.Workspace />,
  hasChildren: true,
  canDrop: () => false,
  TreeContextMenu,
  onSelect: handleSelect,
  haulItems: () => [],
  allowRename: () => true,
  onRename: handleRename,
};
