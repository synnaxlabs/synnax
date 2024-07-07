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
import { useMutation } from "@tanstack/react-query";

import { Menu } from "@/components/menu";
import { useAsyncActionMenu } from "@/hooks/useAsyncAction";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { Ontology } from "@/ontology";
import { Range } from "@/range";
import { create } from "@/schematic/Schematic";
import { type State } from "@/schematic/slice";

const useDelete = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    onMutate: async ({ selection, removeLayout, state: { nodes, setNodes } }) => {
      const ids = selection.resources.map((res) => new ontology.ID(res.key));
      const keys = ids.map((id) => id.key);
      removeLayout(...keys);
      const prevNodes = Tree.deepCopy(nodes);
      const next = Tree.removeNode({
        tree: nodes,
        keys: ids.map((id) => id.toString()),
      });
      setNodes([...next]);
      return prevNodes;
    },
    mutationFn: async ({ client, selection }) => {
      const ids = selection.resources.map((res) => new ontology.ID(res.key));
      await new Promise((resolve) => setTimeout(resolve, 1000));
      await client.workspaces.schematic.delete(ids.map((id) => id.key));
    },
    onError: (err, { state: { setNodes }, addStatus }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      addStatus({
        variant: "error",
        message: "Failed to delete schematic",
        description: err.message,
      });
    },
  }).mutate;

const useCopy = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    mutationFn: async ({
      client,
      selection: { resources, parent },
      state,
      services,
    }) => {
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
    },
  }).mutate;

const useRangeSnapshot = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    mutationFn: async ({ client, selection: { resources, parent }, store }) => {
      const activeRange = Range.selectActiveKey(store.getState());
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
      const rangeID = new ontology.ID({ type: "range", key: activeRange });
      await client.ontology.moveChildren(
        new ontology.ID(parent.key),
        rangeID,
        ...otgsIDs,
      );
    },
  }).mutate;

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { resources },
  } = props;
  const activeRange = Range.useSelect();
  const del = useDelete();
  const copy = useCopy();
  const snapshot = useRangeSnapshot();
  const handleLink = Link.useCopyToClipboard();
  const onSelect = useAsyncActionMenu("schematic.menu", {
    delete: () => del(props),
    copy: () => copy(props),
    rangeSnapshot: () => snapshot(props),
    rename: () => Tree.startRenaming(resources[0].key),
    link: () =>
      handleLink({
        name: resources[0].name,
        resource: resources[0].id.payload,
      }),
  });
  const isSingle = resources.length === 1;
  return (
    <PMenu.Menu onChange={onSelect} level="small" iconSpacing="small">
      <Menu.RenameItem />
      <PMenu.Divider />
      <Menu.DeleteItem />
      <PMenu.Divider />
      {resources.every((r) => r.data?.snapshot === false) && (
        <Range.SnapshotMenuItem range={activeRange} />
      )}
      <PMenu.Item itemKey="copy" startIcon={<Icon.Copy />}>
        Copy
      </PMenu.Item>
      <PMenu.Divider />
      {isSingle && (
        <>
          <Link.CopyMenuItem />
          <PMenu.Divider />
        </>
      )}
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

const handleRename: Ontology.HandleTreeRename = {
  eager: ({ id, name, store }) => store.dispatch(Layout.rename({ key: id.key, name })),
  execute: async ({ client, id, name }) =>
    await client.workspaces.schematic.rename(id.key, name),
  rollback: ({ id, name, store }) =>
    store.dispatch(Layout.rename({ key: id.key, name })),
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
