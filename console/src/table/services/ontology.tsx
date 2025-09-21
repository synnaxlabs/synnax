// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, type Synnax } from "@synnaxlabs/client";
import {
  type Flux,
  Icon,
  Menu as PMenu,
  Mosaic,
  Table as Core,
  Text,
} from "@synnaxlabs/pluto";
import { strings } from "@synnaxlabs/x";
import { useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { Menu } from "@/components";
import { Export } from "@/export";
import { Group } from "@/group";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { Ontology } from "@/ontology";
import { useConfirmDelete } from "@/ontology/hooks";
import { Table } from "@/table";

const useDelete = ({
  state: { getResource },
  selection: { ids },
  removeLayout,
}: Ontology.TreeContextMenuProps): (() => void) => {
  const confirm = useConfirmDelete({ type: "Table" });
  const keys = useMemo(() => ids.map((id) => id.key), [ids]);
  const dispatch = useDispatch();
  const beforeUpdate = useCallback(async () => {
    const ok = await confirm(getResource(ids));
    if (!ok) return false;
    removeLayout(...keys);
    dispatch(Table.remove({ keys }));
    return true;
  }, [confirm, ids]);
  const { update } = Core.useDelete({ beforeUpdate });
  return useCallback(() => update(keys), [update, keys]);
};

const useRename = ({
  selection: {
    ids: [firstID],
  },
  state: { getResource },
}: Ontology.TreeContextMenuProps): (() => void) => {
  const dispatch = useDispatch();
  const beforeUpdate = useCallback(
    async ({ data, rollbacks }: Flux.BeforeUpdateParams<Core.UseRenameArgs>) => {
      const { name: oldName } = data;
      const [name, renamed] = await Text.asyncEdit(ontology.idToString(firstID));
      if (!renamed) return false;
      dispatch(Layout.rename({ key: firstID.key, name }));
      rollbacks.add(() => dispatch(Layout.rename({ key: firstID.key, name: oldName })));
      return { ...data, name };
    },
    [dispatch, firstID],
  );
  const { update } = Core.useRename({ beforeUpdate });
  return useCallback(
    () => update({ key: firstID.key, name: getResource(firstID).name }),
    [update, firstID, getResource],
  );
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { ids, rootID },
    state: { getResource, shape },
  } = props;
  const handleDelete = useDelete(props);
  const handleLink = Cluster.useCopyLinkToClipboard();
  const handleExport = Table.useExport();
  const rename = useRename(props);
  const group = Group.useCreateFromSelection();
  const firstID = ids[0];
  const first = getResource(firstID);
  const onSelect = {
    delete: handleDelete,
    rename,
    link: () => handleLink({ name: first.name, ontologyID: firstID }),
    export: () => handleExport(first.id.key),
    group: () => group(props),
  };
  const isSingle = ids.length === 1;
  return (
    <PMenu.Menu onChange={onSelect} level="small" gap="small">
      <Menu.RenameItem />
      <Menu.DeleteItem />
      <Group.MenuItem ids={ids} shape={shape} rootID={rootID} />
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

const loadTable = async (
  client: Synnax,
  { key }: ontology.ID,
  placeLayout: Layout.Placer,
) => {
  const table = await client.workspaces.tables.retrieve({ key });
  placeLayout(Table.create({ ...table.data, key: table.key, name: table.name }));
};

const handleSelect: Ontology.HandleSelect = ({
  client,
  selection,
  placeLayout,
  handleError,
}) => {
  loadTable(client, selection[0].id, placeLayout).catch((e) => {
    const names = strings.naturalLanguageJoin(
      selection.map(({ name }) => name),
      "table",
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
    const table = await client.workspaces.tables.retrieve({ key });
    placeLayout(
      Table.create({
        name: table.name,
        ...table.data,
        key,
        location: "mosaic",
        tab: { mosaicKey: nodeKey, location },
      }),
    );
  }, "Failed to load table");

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "table",
  icon: <Icon.Table />,
  hasChildren: false,
  onSelect: handleSelect,
  haulItems: ({ id }) => [
    { type: Mosaic.HAUL_CREATE_TYPE, key: ontology.idToString(id) },
  ],
  onMosaicDrop: handleMosaicDrop,
  TreeContextMenu,
};
