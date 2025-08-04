// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, schematic, type Synnax } from "@synnaxlabs/client";
import { Icon, Menu as PMenu, Mosaic, Tree } from "@synnaxlabs/pluto";
import { errors, strings } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";

import { Cluster } from "@/cluster";
import { Menu } from "@/components";
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
      if (!(await confirm(selection.resources))) throw new errors.Canceled();
      const ids = ontology.parseIDs(selection.resources);
      const keys = ids.map((id) => id.key);
      removeLayout(...keys);
      const prevNodes = Tree.deepCopy(nodes);
      const next = Tree.removeNode({
        tree: nodes,
        keys: ids.map((id) => ontology.idToString(id)),
      });
      setNodes([...next]);
      return prevNodes;
    },
    mutationFn: async ({ client, selection }) => {
      const ids = ontology.parseIDs(selection.resources);
      await new Promise((resolve) => setTimeout(resolve, 1000));
      await client.workspaces.schematic.delete(ids.map((id) => id.key));
    },
    onError: (err, { state: { setNodes }, handleError }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      if (errors.Canceled.matches(err)) return;
      handleError(err, "Failed to delete schematic");
    },
  }).mutate;
};

const useCopy = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    mutationFn: async ({
      client,
      selection: { resources, parentID },
      state,
      services,
    }) => {
      if (parentID == null) return;
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
      const otgIDs = schematics.map(({ key }) => schematic.ontologyID(key));
      const otg = await client.ontology.retrieve(otgIDs);
      state.setResources([...state.resources, ...otg]);
      const nextTree = Tree.setNode({
        tree: state.nodes,
        destination: ontology.idToString(parentID),
        additions: Ontology.toTreeNodes(services, otg),
      });
      state.setNodes([...nextTree]);
      Tree.startRenaming(ontology.idToString(otg[0].id));
    },
    onError: (err, { handleError }) => {
      handleError(err, "Failed to copy schematic");
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
  const handleLink = Cluster.useCopyLinkToClipboard();
  const group = Group.useCreateFromSelection();
  const onSelect = useAsyncActionMenu({
    delete: () => del(props),
    copy: () => copy(props),
    rangeSnapshot: () => snapshot(props),
    rename: () => Tree.startRenaming(resources[0].key),
    export: () => handleExport(resources[0].id.key),
    group: () => group(props),
    link: () => handleLink({ name: resources[0].name, ontologyID: resources[0].id }),
  });
  const canEditSchematic = Schematic.useSelectHasPermission();
  const isSingle = resources.length === 1;
  return (
    <PMenu.Menu onChange={onSelect} level="small" gap="small">
      {canEditSchematic && (
        <>
          <Menu.RenameItem />
          <Menu.DeleteItem />
          <Group.MenuItem selection={selection} />
          <PMenu.Divider />
        </>
      )}
      {resources.every((r) => r.data?.snapshot === false) && (
        <>
          <Range.SnapshotMenuItem range={activeRange} />
          <PMenu.Item itemKey="copy" startIcon={<Icon.Copy />}>
            Copy
          </PMenu.Item>
          <PMenu.Divider />
        </>
      )}
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

const handleSelect: Ontology.HandleSelect = ({
  client,
  selection,
  placeLayout,
  handleError,
}) => {
  loadSchematic(client, selection[0].id, placeLayout).catch((e) => {
    const names = strings.naturalLanguageJoin(
      selection.map(({ name }) => name),
      "schematic",
    );
    handleError(e, `Failed to select ${names}`);
  });
};

const handleMosaicDrop: Ontology.HandleMosaicDrop = ({
  client,
  id,
  location,
  nodeKey,
  placeLayout,
  handleError,
}) =>
  handleError(async () => {
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
  }, "Failed to load schematic");

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: schematic.ONTOLOGY_TYPE,
  icon: <Icon.Schematic />,
  hasChildren: false,
  onSelect: handleSelect,
  haulItems: ({ id }) => [
    { type: Mosaic.HAUL_CREATE_TYPE, key: ontology.idToString(id) },
  ],
  allowRename: () => true,
  onRename: handleRename,
  onMosaicDrop: handleMosaicDrop,
  TreeContextMenu,
};
