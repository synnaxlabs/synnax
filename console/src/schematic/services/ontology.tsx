// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, ranger, schematic, type Synnax } from "@synnaxlabs/client";
import {
  ContextMenu as PContextMenu,
  type Flux,
  Icon,
  Mosaic,
  Schematic as Core,
  Status,
  Text,
} from "@synnaxlabs/pluto";
import { array, strings } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Cluster } from "@/cluster";
import { ContextMenu } from "@/components";
import { Export } from "@/export";
import { Group } from "@/group";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { Ontology } from "@/ontology";
import { createUseDelete } from "@/ontology/createUseDelete";
import { createUseRename } from "@/ontology/createUseRename";
import { Range } from "@/range";
import { Schematic } from "@/schematic";

const useDelete = createUseDelete({
  type: "Schematic",
  query: Core.useDelete,
  convertKey: String,
  beforeUpdate: async ({ data, removeLayout, store }) => {
    removeLayout(...data);
    store.dispatch(Schematic.remove({ keys: array.toArray(data) }));
    return data;
  },
});

const useCopy = (props: Ontology.TreeContextMenuProps): (() => void) => {
  const {
    selection: { ids },
    state: { getResource },
  } = props;
  const rename = Core.useRename();
  const copy = Core.useCopy({
    afterSuccess: useCallback(
      async ({ data }: Flux.AfterSuccessParams<schematic.Schematic>) => {
        const id = schematic.ontologyID(data.key);
        const [name, renamed] = await Text.asyncEdit(ontology.idToString(id));
        if (!renamed) return;
        await rename.updateAsync({ key: data.key, name });
      },
      [rename],
    ),
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
    ({ schematics }: Core.SnapshotParams) =>
      `${strings.naturalLanguageJoin(
        array.toArray(schematics).map((s) => s.name),
        "schematic",
      )} to ${rng?.name ?? "active range"}`,
    [rng],
  );
  const { update } = Core.useSnapshot({
    afterSuccess: useCallback(
      ({ data }: Flux.AfterSuccessParams<Core.SnapshotParams>) =>
        addStatus({
          variant: "success",
          message: `Successfully snapshotted ${buildMessage(data)}`,
        }),
      [buildMessage, addStatus],
    ),
    afterFailure: ({ status, data }: Flux.AfterFailureParams<Core.SnapshotParams>) =>
      addStatus({ ...status, message: `Failed to snapshot ${buildMessage(data)}` }),
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

const useRename = createUseRename({
  query: Core.useRename,
  ontologyID: schematic.ontologyID,
  convertKey: String,
  beforeUpdate: async ({ data, rollbacks, store, oldName }) => {
    const { key, name } = data;
    store.dispatch(Layout.rename({ key, name }));
    rollbacks.push(() => store.dispatch(Layout.rename({ key, name: oldName })));
    return { ...data, name };
  },
});

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { ids },
    state: { getResource },
  } = props;
  const activeRange = Range.useSelect();
  const handleDelete = useDelete(props);
  const handleCopy = useCopy(props);
  const snapshot = useRangeSnapshot();
  const exportSchematic = Schematic.useExport();
  const copyLink = Cluster.useCopyLinkToClipboard();
  const handleRename = useRename(props);
  const firstID = ids[0];
  const resources = getResource(ids);
  const first = resources[0];
  const canEditSchematic = Schematic.useSelectHasPermission();
  const isSingle = ids.length === 1;
  const handleRangeSnapshot = () => snapshot(props);
  const handleExport = () => exportSchematic(first.id.key);
  const handleLink = () => copyLink({ name: first.name, ontologyID: firstID });
  return (
    <>
      {canEditSchematic && (
        <>
          <ContextMenu.RenameItem onClick={handleRename} />
          <ContextMenu.DeleteItem onClick={handleDelete} />
          <Group.ContextMenuItem {...props} />
          <PContextMenu.Divider />
        </>
      )}
      {resources.every((r) => r.data?.snapshot === false) && (
        <>
          <Range.SnapshotContextMenuItem
            range={activeRange}
            onClick={handleRangeSnapshot}
          />
          <PContextMenu.Item onClick={handleCopy}>
            <Icon.Copy />
            Copy
          </PContextMenu.Item>
          <PContextMenu.Divider />
        </>
      )}
      {isSingle && (
        <>
          <Export.ContextMenuItem onClick={handleExport} />
          <Link.CopyContextMenuItem onClick={handleLink} />
          <Ontology.CopyContextMenuItem {...props} />
          <PContextMenu.Divider />
        </>
      )}
      <ContextMenu.ReloadConsoleItem />
    </>
  );
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
  onMosaicDrop: handleMosaicDrop,
  TreeContextMenu,
};
