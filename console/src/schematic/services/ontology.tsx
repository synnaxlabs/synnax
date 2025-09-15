// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, schematic, type Synnax } from "@synnaxlabs/client";
import {
  Icon,
  Menu as PMenu,
  Mosaic,
  Schematic as Core,
  Text,
  Tree,
} from "@synnaxlabs/pluto";
import { strings } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { useCallback } from "react";

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

const useDelete = (props: Ontology.TreeContextMenuProps): (() => void) => {
  const {
    state: { getResource },
    selection: { ids: ids },
  } = props;
  const confirm = useConfirmDelete({ type: "Schematic" });
  const beforeUpdate = useCallback(
    () => confirm(getResource(ids)),
    [confirm, ids],
  );
  const { update } = Core.useDelete({ beforeUpdate });
  return useCallback(
    () => update(ids.map((id) => id.key)),
    [update, ids],
  );
};

const useCopy = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    mutationFn: async ({
      client,
      selection: { ids: ids, parentID },
      state: { nodes, setNodes, setResource, getResource },
    }) => {
      if (parentID == null) return;
      const schematics = await Promise.all(
        ids.map(
          async (id) =>
            await client.workspaces.schematics.copy(
              id.key,
              `${getResource(id).name} (copy)`,
              false,
            ),
        ),
      );
      const otgIDs = schematics.map(({ key }) => schematic.ontologyID(key));
      const otg = await client.ontology.retrieve(otgIDs);
      setResource(otg);
      const nextTree = Tree.setNode({
        tree: nodes,
        destination: ontology.idToString(parentID),
        additions: otg,
      });
      setNodes([...nextTree]);
      Text.edit(ontology.idToString(otg[0].id));
    },
    onError: (err, { handleError }) => {
      handleError(err, "Failed to copy schematic");
    },
  }).mutate;

const useSnapshot = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const snapshot = Schematic.useRangeSnapshot();
  return ({ selection: { ids: ids }, state: { getResource } }) => {
    const schematics = ids.map((id) => ({
      key: id.key,
      name: getResource(id).name,
    }));
    snapshot(schematics);
  };
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { ids: ids, rootID },
    state: { getResource, shape },
  } = props;
  const activeRange = Range.useSelect();
  const handleDelete = useDelete(props);
  const copy = useCopy();
  const snapshot = useSnapshot();
  const handleExport = Schematic.useExport();
  const handleLink = Cluster.useCopyLinkToClipboard();
  const group = Group.useCreateFromSelection();
  const firstID = ids[0];
  const resources = getResource(ids);
  const first = resources[0];
  const onSelect = useAsyncActionMenu({
    delete: handleDelete,
    copy: () => copy(props),
    rangeSnapshot: () => snapshot(props),
    rename: () => Text.edit(ontology.idToString(firstID)),
    export: () => handleExport(first.id.key),
    group: () => group(props),
    link: () => handleLink({ name: first.name, ontologyID: firstID }),
  });
  const canEditSchematic = Schematic.useSelectHasPermission();
  const isSingle = ids.length === 1;
  return (
    <PMenu.Menu onChange={onSelect} level="small" gap="small">
      {canEditSchematic && (
        <>
          <Menu.RenameItem />
          <Menu.DeleteItem />
          <Group.MenuItem ids={ids} shape={shape} rootID={rootID} />
          <PMenu.Divider />
        </>
      )}
      {resources.every((r) => r.data?.snapshot === false) && (
        <>
          <Range.SnapshotMenuItem range={activeRange} />
          <PMenu.Item itemKey="copy">
            <Icon.Copy />
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
    await client.workspaces.schematics.rename(id.key, name),
  rollback: ({ id: { key }, name, store }) =>
    store.dispatch(Layout.rename({ key, name })),
};

const loadSchematic = async (
  client: Synnax,
  { key }: ontology.ID,
  placeLayout: Layout.Placer,
) => {
  const schematic = await client.workspaces.schematics.retrieve({ key });
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
  id: { key },
  location,
  nodeKey,
  placeLayout,
  handleError,
}) =>
  handleError(async () => {
    const schematic = await client.workspaces.schematics.retrieve({ key });
    placeLayout(
      Schematic.create({
        name: schematic.name,
        ...schematic.data,
        key,
        location: "mosaic",
        tab: { mosaicKey: nodeKey, location },
      }),
    );
  }, "Failed to load schematic");

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "schematic",
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
