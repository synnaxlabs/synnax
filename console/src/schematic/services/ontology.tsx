// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, ranger, type Synnax } from "@synnaxlabs/client";
import {
  type Flux,
  Icon,
  Menu as PMenu,
  Mosaic,
  Schematic as Core,
  Status,
  Text,
} from "@synnaxlabs/pluto";
import { array, strings } from "@synnaxlabs/x";
import { useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

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

const useDelete = ({
  state: { getResource },
  selection: { ids },
}: Ontology.TreeContextMenuProps): (() => void) => {
  const confirm = useConfirmDelete({ type: "Schematic" });
  const keys = useMemo(() => ids.map((id) => id.key), [ids]);
  const dispatch = useDispatch();
  const beforeUpdate = useCallback(async () => {
    const ok = await confirm(getResource(ids));
    if (!ok) return false;
    dispatch(Schematic.remove({ keys }));
    return true;
  }, [confirm, ids, getResource, dispatch]);
  const { update } = Core.useDelete({ beforeUpdate });
  return useCallback(() => update(keys), [update, keys]);
};

const useCopy = ({
  selection: { ids },
  state: { getResource },
}: Ontology.TreeContextMenuProps): (() => void) => {
  const copy = Core.useCopy({
    afterSuccess: async () => Text.edit(ontology.idToString(ids[0])),
  });
  return () =>
    ids.map((id) => {
      const name = `${getResource(id).name} (copy)`;
      copy.update({ key: id.key, name, snapshot: false });
    });
};

export const useRangeSnapshot = () => {
  const addStatus = Status.useAdder();
  const rng = Range.useSelect();
  const buildMessage = useCallback(
    ({ schematics }: Core.UseSnapshotArgs) =>
      `${strings.naturalLanguageJoin(
        array.toArray(schematics).map((s) => s.name),
        "schematic",
      )} to ${rng?.name ?? "active range"}`,
    [rng],
  );
  const { update } = Core.useCreateSnapshot({
    afterSuccess: useCallback(
      ({ value }: Flux.AfterSuccessArgs<Core.UseSnapshotArgs>) =>
        addStatus({
          variant: "success",
          message: `Successfully snapshotted ${buildMessage(value)}`,
        }),
      [buildMessage, addStatus],
    ),
    afterFailure: ({ status, value }: Flux.AfterFailureArgs<Core.UseSnapshotArgs>) =>
      addStatus({ ...status, message: `Failed to snapshot ${buildMessage(value)}` }),
  });
  return ({
    selection: { ids },
    state: { getResource },
  }: Ontology.TreeContextMenuProps) => {
    if (rng == null)
      return addStatus({
        variant: "error",
        message: "Cannot snapshot schematics without an active range",
      });
    const schematics = ids.map((id) => ({
      key: id.key,
      name: getResource(id).name,
    }));
    const parentID = ranger.ontologyID(rng.key);
    update({ schematics, parentID });
  };
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { ids, rootID },
    state: { getResource, shape },
  } = props;
  const activeRange = Range.useSelect();
  const handleDelete = useDelete(props);
  const handleCopy = useCopy(props);
  const snapshot = useRangeSnapshot();
  const handleExport = Schematic.useExport();
  const handleLink = Cluster.useCopyLinkToClipboard();
  const group = Group.useCreateFromSelection();
  const firstID = ids[0];
  const resources = getResource(ids);
  const first = resources[0];
  const onSelect = useAsyncActionMenu({
    delete: handleDelete,
    copy: handleCopy,
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
