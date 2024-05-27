// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Menu } from "@synnaxlabs/pluto";
import { Tree } from "@synnaxlabs/pluto/tree";
import { type UnknownRecord, deep } from "@synnaxlabs/x";

import { Group } from "@/group";
import { Layout } from "@/layout";
import { Ontology } from "@/ontology";
import { Schematic } from "@/schematic";
import { selectActiveKey } from "@/workspace/selectors";
import { add, rename, setActive } from "@/workspace/slice";

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
      store.dispatch(Layout.clearWorkspace());
    }
    await client.workspaces.delete(...resources.map((r) => r.id.key));
    const next = Tree.removeNode({
      tree: nodes,
      keys: resources.map((r) => r.id.toString()),
    });
    setNodes([...next]);
  })();
};

const handleCreateNewSchematic = ({
  store,
  client,
  services,
  placeLayout,
  selection,
  state: { nodes, setNodes, resources, setResources },
}: Ontology.TreeContextMenuProps): void => {
  const ws = selection.resources[0].id.key;
  void (async () => {
    const schematic = await client.workspaces.schematic.create(ws, {
      name: "New Schematic",
      snapshot: false,
      data: deep.copy(Schematic.ZERO_STATE) as unknown as UnknownRecord,
    });
    const otg = await client.ontology.retrieve(
      new ontology.ID({ key: schematic.key, type: "schematic" }),
    );
    placeLayout(
      Schematic.create({
        ...(schematic.data as unknown as Schematic.State),
        key: schematic.key,
        name: schematic.name,
        snapshot: schematic.snapshot,
      }),
    );
    setResources([...resources, otg]);
    const nextNodes = Tree.setNode({
      tree: nodes,
      destination: selection.resources[0].key,
      additions: Ontology.toTreeNodes(services, [otg]),
    });
    setNodes([...nextNodes]);
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
        return;
      case "schematic": {
        return handleCreateNewSchematic(props);
      }
    }
  };

  return (
    <Menu.Menu onChange={handleSelect} level="small" iconSpacing="small">
      <Menu.Item itemKey="delete" startIcon={<Icon.Delete />}>
        Delete
      </Menu.Item>
      <Ontology.RenameMenuItem />
      <Group.GroupMenuItem selection={props.selection} />
      <Menu.Item itemKey="plot" startIcon={<Icon.Visualize />}>
        New Line Plot
      </Menu.Item>
      <Menu.Item itemKey="schematic" startIcon={<Icon.Schematic />}>
        New Schematic
      </Menu.Item>
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
      }),
    );
  })();
};

const handleRename: Ontology.HandleTreeRename = ({
  client,
  id,
  name,
  store,
  state: { nodes, setNodes },
}) => {
  void client.workspaces.rename(id.key, name);
  store.dispatch(rename({ key: id.key, name }));

  const next = Tree.updateNode({
    tree: nodes,
    key: id.toString(),
    updater: (node) => ({ ...node, name }),
  });
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
