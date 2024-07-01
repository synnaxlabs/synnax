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

import { Menu } from "@/components/menu";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { Ontology } from "@/ontology";
import { Range } from "@/range";
import { create, type State } from "@/schematic/slice";

const TreeContextMenu: Ontology.TreeContextMenu = ({
  client,
  removeLayout,
  store,
  services,
  selection: { resources, parent },
  state,
}) => {
  const ids = resources.map((res) => new ontology.ID(res.key));
  const keys = ids.map((id) => id.key);
  const activeRange = Range.select(store.getState());

  const handleDelete = (): void => {
    void (async () => {
      await client.workspaces.schematic.delete(keys);
      removeLayout(...keys);
      const next = Tree.removeNode({
        tree: state.nodes,
        keys: ids.map((id) => id.toString()),
      });
      state.setNodes([...next]);
    })();
  };

  const handleCopy = (): void => {
    void (async () => {
      if (parent == null) return;
      const schematics = await Promise.all(
        resources.map(
          async (res) =>
            await client.workspaces.schematic.copy(
              res.id.key,
              res.name + " (copy)",
              false,
            ),
        ),
      );
      const otgIDs = schematics.map(
        ({ key }) => new ontology.ID({ type: "schematic", key }),
      );
      const otg = await client.ontology.retrieve(otgIDs);
      state.setResources([...state.resources, ...otg]);
      const nextTree = Tree.setNode({
        tree: state.nodes,
        destination: parent.key,
        additions: Ontology.toTreeNodes(services, otg),
      });
      state.setNodes([...nextTree]);
      Tree.startRenaming(otg[0].id.toString());
    })();
  };

  const handleRangeSnapshot = (): void => {
    void (async () => {
      if (activeRange == null || parent == null) return;
      const schematics = await Promise.all(
        resources.map(
          async (res) =>
            await client.workspaces.schematic.copy(
              res.id.key,
              res.name + " (snap)",
              true,
            ),
        ),
      );
      const otgsIDs = schematics.map(
        ({ key }) => new ontology.ID({ type: "schematic", key }),
      );
      const rangeID = new ontology.ID({ type: "range", key: activeRange.key });
      await client.ontology.moveChildren(
        new ontology.ID(parent.key),
        rangeID,
        ...otgsIDs,
      );
    })();
  };

  const handleRename = (): void => Tree.startRenaming(resources[0].key);

  const handleCopyUrl = (): void => {
    const url = `synnax://cluster/${client.key}/schematic/${resources[0].id.key}`;
    void navigator.clipboard.writeText(url);
  };

  const f: Record<string, () => void> = {
    delete: handleDelete,
    rename: handleRename,
    copy: handleCopy,
    rangeSnapshot: handleRangeSnapshot,
    link: handleCopyUrl,
  };

  const onSelect = (key: string): void => f[key]?.();
  const isSingle = resources.length === 1;

  return (
    <PMenu.Menu onChange={onSelect} level="small" iconSpacing="small">
      <Menu.RenameItem />
      {resources.every((r) => r.data?.snapshot === false) && (
        <Range.SnapshotMenuItem range={activeRange} />
      )}
      <PMenu.Item itemKey="copy" startIcon={<Icon.Copy />}>
        Copy
      </PMenu.Item>
      <Menu.DeleteItem />
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
  state: { nodes, setNodes, resources, setResources },
}) => {
  void client.workspaces.schematic.rename(id.key, name);
  store.dispatch(Layout.rename({ key: id.key, name }));
  const next = Tree.updateNode({
    tree: nodes,
    key: id.toString(),
    updater: (node) => ({ ...node, name }),
  });
  setResources([
    ...resources.map((res) => ({
      ...res,
      name: res.id.toString() === id.toString() ? name : res.name,
    })),
  ]);
  setNodes([...next]);
};

const handleSelect: Ontology.HandleSelect = ({ client, selection, placeLayout }) => {
  void (async () => {
    const schematic = await client.workspaces.schematic.retrieve(selection[0].id.key);
    placeLayout(
      create({
        ...(schematic.data as unknown as State),
        key: schematic.key,
        name: schematic.name,
        snapshot: schematic.snapshot,
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
    const schematic = await client.workspaces.schematic.retrieve(id.key);
    placeLayout(
      create({
        name: schematic.name,
        ...(schematic.data as unknown as State),
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
  type: "schematic",
  icon: <Icon.Schematic />,
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
