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
import { Menu as PMenu, Mosaic, Tree } from "@synnaxlabs/pluto";

import { Cluster } from "@/cluster";
import { Menu } from "@/components/menu";
import { Layout } from "@/layout";
import { create, type State } from "@/lineplot/slice";
import { Link } from "@/link";
import { Ontology } from "@/ontology";

const TreeContextMenu: Ontology.TreeContextMenu = ({
  client,
  removeLayout,
  selection: { resources },
  state: { nodes, setNodes },
}) => {
  const ids = resources.map((res) => new ontology.ID(res.key));
  const keys = ids.map((id) => id.key);
  const handleDelete = (): void => {
    void (async () => {
      await client.workspaces.linePlot.delete(keys);
      removeLayout(...keys);
      const next = Tree.removeNode({
        tree: nodes,
        keys: ids.map((id) => id.toString()),
      });
      setNodes([...next]);
    })();
  };

  const handleRename = (): void => Tree.startRenaming(resources[0].key);

  const clusterKey = Cluster.useSelectActiveKey();
  const handleCopyLink = (): void => {
    const toCopy = `synnax://cluster/${clusterKey}/lineplot/${resources[0].id.key}`;
    void navigator.clipboard.writeText(toCopy);
  };

  const f: Record<string, () => void> = {
    delete: handleDelete,
    rename: handleRename,
    link: handleCopyLink,
  };

  const onSelect = (key: string): void => f[key]();

  const isSingle = resources.length === 1;

  return (
    <PMenu.Menu onChange={onSelect} level="small" iconSpacing="small">
      {isSingle && <Menu.RenameItem />}
      <PMenu.Item itemKey="delete" startIcon={<Icon.Delete />}>
        Delete
      </PMenu.Item>
      {isSingle && <Link.CopyMenuItem />}
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

const handleRename: Ontology.HandleTreeRename = ({
  client,
  id,
  name,
  store,
  state: { nodes, setNodes },
}) => {
  void client.workspaces.linePlot.rename(id.key, name);
  store.dispatch(Layout.rename({ key: id.key, name }));
  const next = Tree.updateNode({
    tree: nodes,
    key: id.toString(),
    updater: (node) => ({ ...node, name }),
  });
  setNodes([...next]);
};

const handleSelect: Ontology.HandleSelect = ({ client, selection, placeLayout }) => {
  void (async () => {
    const linePlot = await client.workspaces.linePlot.retrieve(selection[0].id.key);
    placeLayout(
      create({
        ...(linePlot.data as unknown as State),
        key: linePlot.key,
        name: linePlot.name,
      }),
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
    const linePlot = await client.workspaces.linePlot.retrieve(id.key);
    placeLayout(
      create({
        ...(linePlot.data as unknown as State),
        key: linePlot.key,
        name: linePlot.name,
        location: "mosaic",
        tab: {
          mosaicKey: nodeKey,
          location,
        },
      }),
    );
  })();
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: "lineplot",
  icon: <Icon.Visualize />,
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
