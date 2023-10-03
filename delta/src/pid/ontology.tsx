// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Menu, Mosaic, Tree } from "@synnaxlabs/pluto";

import { Layout } from "@/layout";
import { Ontology } from "@/ontology";
import { create, type State } from "@/pid/slice";

const TreeContextMenu: Ontology.TreeContextMenu = ({
  client,
  store,
  removeLayout,
  selection: { resources },
  state: { nodes, setNodes },
}) => {
  const ids = resources.map((res) => new ontology.ID(res.key));
  const keys = ids.map((id) => id.key);
  const handleDelete = (): void => {
    void (async () => {
      await client.workspaces.pid.delete(keys);
      removeLayout(...keys);
      const next = Tree.removeNode(nodes, ...ids.map((id) => id.toString()));
      setNodes([...next]);
    })();
  };

  const handleRename = (): void => Tree.startRenaming(resources[0].key);

  const f: Record<string, () => void> = {
    delete: handleDelete,
    rename: handleRename,
  };

  const onSelect = (key: string): void => f[key]();

  return (
    <Menu.Menu onChange={onSelect} level="small" iconSpacing="small">
      <Ontology.RenameMenuItem />
      <Menu.Item itemKey="delete" startIcon={<Icon.Delete />}>
        Delete
      </Menu.Item>
    </Menu.Menu>
  );
};

const handleRename: Ontology.HandleTreeRename = ({
  client,
  id,
  name,
  store,
  state: { nodes, setNodes },
}) => {
  void client.workspaces.pid.rename(id.key, name);
  store.dispatch(Layout.rename({ key: id.key, name }));
  const next = Tree.updateNode(nodes, id.toString(), (node) => ({ ...node, name }));
  setNodes([...next]);
};

const handleSelect: Ontology.HandleSelect = ({
  client,
  selection,
  store,
  placeLayout,
}) => {
  void (async () => {
    const pid = await client.workspaces.pid.retrieve(selection[0].id.key);
    placeLayout(
      create({ key: pid.key, name: pid.name, ...(pid.data as unknown as State) })
    );
  })();
};

const handleMosaicDrop: Ontology.HandleMosaicDrop = ({
  client,
  id,
  location,
  nodeKey,
  placeLayout,
}) => {
  void (async () => {
    const pid = await client.workspaces.pid.retrieve(id.key);
    placeLayout(
      create({
        ...(pid.data as unknown as State),
        location: "mosaic",
        tab: {
          mosaicKey: nodeKey,
          location,
        },
      })
    );
  })();
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: "pid",
  icon: <Icon.PID />,
  hasChildren: false,
  haulItems: (r) => [
    {
      type: Mosaic.HAUL_CREATE_TYPE,
      key: r.id.toString(),
    },
  ],
  allowRename: () => true,
  onRename: handleRename,
  canDrop: () => false,
  TreeContextMenu,
  onMosaicDrop: handleMosaicDrop,
  onSelect: handleSelect,
};
