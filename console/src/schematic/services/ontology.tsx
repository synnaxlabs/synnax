// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, type Synnax } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Menu as PMenu, Mosaic, Tree } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";

import { Menu } from "@/components/menu";
import { Export } from "@/export";
import { Group } from "@/group";
import { useAsyncActionMenu } from "@/hooks/useAsyncAction";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { Ontology } from "@/ontology";
import { useConfirmDelete } from "@/ontology/hooks";
import { Range } from "@/range";
import { Schematic } from "@/schematic";

const useDelete = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const confirm = useConfirmDelete({ type: "Schematic" });
  return useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    onMutate: async ({ selection, removeLayout, state: { nodes, setNodes } }) => {
      if (!(await confirm(selection.resources))) throw errors.CANCELED;
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
    onError: (err, { state: { setNodes }, handleException }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      if (errors.CANCELED.matches(err)) return;
      handleException(err, "Failed to delete schematic");
    },
  }).mutate;
};

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
              `${res.name} (copy)`,
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
    onError: (err, { handleException }) => {
      handleException(err, "Failed to copy schematic");
    },
  }).mutate;

const useSnapshot = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const snapshot = Schematic.useRangeSnapshot();
  return ({ selection: { resources } }) => {
    const schematics = resources.map((res) => ({ key: res.id.key, name: res.name }));
    snapshot(schematics);
  };
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection,
    selection: { resources },
  } = props;
  const activeRange = Range.useSelect();
  const del = useDelete();
  const copy = useCopy();
  const snapshot = useSnapshot();
  const handleExport = Schematic.useExport();
  const handleLink = Link.useCopyToClipboard();
  const group = Group.useCreateFromSelection();
  const onSelect = useAsyncActionMenu({
    delete: () => del(props),
    copy: () => copy(props),
    rangeSnapshot: () => snapshot(props),
    rename: () => Tree.startRenaming(resources[0].key),
    export: () => handleExport(resources[0].id.key),
    group: () => group(props),
    link: () =>
      handleLink({ name: resources[0].name, ontologyID: resources[0].id.payload }),
  });
  const canEditSchematic = Schematic.useSelectHasPermission();
  const isSingle = resources.length === 1;
  return (
    <PMenu.Menu onChange={onSelect} level="small" iconSpacing="small">
      {canEditSchematic && (
        <>
          <Menu.RenameItem />
          <Menu.DeleteItem />
          <Group.GroupMenuItem selection={selection} />
          <PMenu.Divider />
        </>
      )}
      {resources.every((r) => r.data?.snapshot === false) && (
        <Range.SnapshotMenuItem range={activeRange} />
      )}
      <PMenu.Item itemKey="copy" startIcon={<Icon.Copy />}>
        Copy
      </PMenu.Item>
      <PMenu.Divider />
      {isSingle && (
        <>
          <Export.MenuItem />
          <Link.CopyMenuItem />
          <PMenu.Divider />
        </>
      )}
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

const handleRename: Ontology.HandleTreeRename = {
  eager: ({ id: { key }, name, store }) => store.dispatch(Layout.rename({ key, name })),
  execute: async ({ client, id, name }) =>
    await client.workspaces.schematic.rename(id.key, name),
  rollback: ({ id: { key }, name, store }) =>
    store.dispatch(Layout.rename({ key, name })),
};

const loadSchematic = async (
  client: Synnax,
  id: ontology.ID,
  placeLayout: Layout.Placer,
) => {
  const schematic = await client.workspaces.schematic.retrieve(id.key);
  placeLayout(
    Schematic.create({
      ...schematic.data,
      key: schematic.key,
      name: schematic.name,
      snapshot: schematic.snapshot,
      editable: false,
    }),
  );
};

const handleSelect: Ontology.HandleSelect = async ({
  client,
  selection,
  placeLayout,
}) => await loadSchematic(client, selection[0].id, placeLayout);

const handleMosaicDrop: Ontology.HandleMosaicDrop = ({
  client,
  id,
  location,
  nodeKey,
  placeLayout,
  handleException,
}) => {
  void (async () => {
    try {
      const schematic = await client.workspaces.schematic.retrieve(id.key);
      placeLayout(
        Schematic.create({
          name: schematic.name,
          ...schematic.data,
          key: id.key,
          location: "mosaic",
          tab: { mosaicKey: nodeKey, location },
        }),
      );
    } catch (e) {
      handleException(e, "Failed to load schematic");
    }
  })();
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: "schematic",
  icon: <Icon.Schematic />,
  hasChildren: false,
  haulItems: (r) => [{ type: Mosaic.HAUL_CREATE_TYPE, key: r.id.toString() }],
  allowRename: () => true,
  onRename: handleRename,
  canDrop: () => false,
  TreeContextMenu,
  onMosaicDrop: handleMosaicDrop,
  onSelect: handleSelect,
};
