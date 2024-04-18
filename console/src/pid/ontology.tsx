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
import { Range } from "@/range";

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
      await client.workspaces.pid.delete(keys);
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
      const pids = await Promise.all(
        resources.map(
          async (res) =>
            await client.workspaces.pid.copy(res.id.key, res.name + " (copy)", false),
        ),
      );
      const otgIDs = pids.map(({ key }) => new ontology.ID({ type: "pid", key }));
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
      if (activeRange == null) return;
      const pids = await Promise.all(
        resources.map(
          async (res) =>
            await client.workspaces.pid.copy(res.id.key, res.name + " (snap)", true),
        ),
      );
      const otgsIDs = pids.map(({ key }) => new ontology.ID({ type: "pid", key }));
      const rangeID = new ontology.ID({ type: "range", key: activeRange.key });
      await client.ontology.moveChildren(
        new ontology.ID(parent.key),
        rangeID,
        ...otgsIDs,
      );
    })();
  };

  const handleRename = (): void => Tree.startRenaming(resources[0].key);

  const f: Record<string, () => void> = {
    delete: handleDelete,
    rename: handleRename,
    copy: handleCopy,
    rangeSnapshot: handleRangeSnapshot,
  };

  const onSelect = (key: string): void => f[key]();

  return (
    <Menu.Menu onChange={onSelect} level="small" iconSpacing="small">
      <Ontology.RenameMenuItem />
      {resources.every((r) => r.data?.snapshot === false) && (
        <Range.SnapshotMenuItem range={activeRange} />
      )}
      <Menu.Item itemKey="copy" startIcon={<Icon.Copy />}>
        Copy
      </Menu.Item>
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
  state: { nodes, setNodes, resources, setResources },
}) => {
  void client.workspaces.pid.rename(id.key, name);
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
    const pid = await client.workspaces.pid.retrieve(selection[0].id.key);
    placeLayout(
      create({
        ...(pid.data as unknown as State),
        key: pid.key,
        name: pid.name,
        snapshot: pid.snapshot,
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
    const pid = await client.workspaces.pid.retrieve(id.key);
    placeLayout(
      create({
        name: pid.name,
        ...(pid.data as unknown as State),
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
