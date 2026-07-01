// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, ranger, schematic, type Synnax } from "@synnaxlabs/client";
import {
  Access,
  type Flux,
  Icon,
  Menu,
  Mosaic,
  Schematic as Base,
  Status,
  Text,
} from "@synnaxlabs/pluto";
import { array, strings } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Cluster } from "@/component/cluster";
import { ContextMenu } from "@/component/context-menu";
import { Export } from "@/component/export";
import { Range } from "@/component/range";
import { Schematic } from "@/component/schematic";
import { Group } from "@/service/group";
import { Link } from "@/service/link";
import { Ontology } from "@/service/ontology";
import { useExport } from "@/service/schematic/export";
import { Session } from "@/session";

const useDelete = Ontology.createUseDelete({
  type: "Schematic",
  query: Base.useDelete,
  convertKey: String,
  beforeUpdate: async ({ data, removeLayout, store }) => {
    removeLayout(...data);
    store.dispatch(Session.Schematic.remove({ keys: array.toArray(data) }));
    return data;
  },
});

const useCopy = (props: Ontology.TreeContextMenuProps): (() => void) => {
  const {
    selection: { ids },
    state: { getResource },
  } = props;
  const rename = Base.useRename();
  const copy = Base.useCopy({
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
    ({ schematics }: Base.SnapshotParams) =>
      `${strings.naturalLanguageJoin(
        array.toArray(schematics).map((s) => s.name),
        "schematic",
      )} to ${rng?.name ?? "active range"}`,
    [rng],
  );
  const { update } = Base.useSnapshot({
    afterSuccess: useCallback(
      ({ data }: Flux.AfterSuccessParams<Base.SnapshotParams>) =>
        addStatus({
          variant: "success",
          message: `Successfully snapshotted ${buildMessage(data)}`,
        }),
      [buildMessage, addStatus],
    ),
    afterFailure: ({ status, data }: Flux.AfterFailureParams<Base.SnapshotParams>) =>
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

const useRename = Ontology.createUseRename({
  query: Base.useRename,
  ontologyID: schematic.ontologyID,
  convertKey: String,
  beforeUpdate: async ({ data, rollbacks, store, oldName }) => {
    const { key, name } = data;
    store.dispatch(Session.Layout.rename({ key, name }));
    rollbacks.push(() => store.dispatch(Session.Layout.rename({ key, name: oldName })));
    return { ...data, name };
  },
});

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { ids, rootID },
    state: { getResource, shape },
  } = props;
  const activeRange = Range.useSelect();
  const hasCreatePermission = Access.useCreateGranted(schematic.TYPE_ONTOLOGY_ID);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const handleDelete = useDelete(props);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const handleCopy = useCopy(props);
  const snapshot = useRangeSnapshot();
  const handleExport = useExport();
  const handleLink = Cluster.useCopyLinkToClipboard();
  const rename = useRename(props);
  const group = Group.useCreateFromSelection();
  const firstID = ids[0];
  const resources = getResource(ids);
  const first = resources[0];
  return (
    <ContextMenu.Menu>
      {hasDeletePermission && <ContextMenu.DeleteItem onClick={handleDelete} />}
      {hasUpdatePermission && (
        <>
          <ContextMenu.RenameItem onClick={rename} />
          <Group.ContextMenuItem
            ids={ids}
            shape={shape}
            rootID={rootID}
            onClick={() => group(props)}
          />
          <Menu.Divider />
        </>
      )}
      {resources.every((r) => r.data?.snapshot === false) && hasCreatePermission && (
        <>
          <Range.SnapshotMenuItem range={activeRange} onClick={() => snapshot(props)} />
          <Menu.Item itemKey="copy" onClick={handleCopy}>
            <Icon.Copy />
            Copy
          </Menu.Item>
          <Menu.Divider />
        </>
      )}
      <Export.ContextMenuItem onClick={() => handleExport(first.id.key)} />
      <Link.CopyContextMenuItem
        onClick={() => handleLink({ name: first.name, ontologyID: firstID })}
      />
      <Ontology.CopyPropertiesContextMenuItem {...props} />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const loadSchematic = async (
  client: Synnax,
  { key }: ontology.ID,
  placeLayout: Session.Layout.Placer,
) => {
  const schematic = await client.schematics.retrieve({ key });
  placeLayout(Schematic.create({ key: schematic.key, name: schematic.name }));
};

const handleSelect: Ontology.HandleSelect = ({
  client,
  selection,
  placeLayout,
  handleError,
}) => {
  loadSchematic(client, selection[0].id, placeLayout).catch((e: unknown) => {
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
    const schematic = await client.schematics.retrieve({ key });
    placeLayout(
      Schematic.create({
        key: schematic.key,
        name: schematic.name,
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
  haulItems: ({ id }) => [Mosaic.createTabCreateHaulItem(ontology.idToString(id))],
  onMosaicDrop: handleMosaicDrop,
  TreeContextMenu,
};
