// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { Icon, LinePlot as Core, Menu as PMenu, Mosaic, Text } from "@synnaxlabs/pluto";
import { strings } from "@synnaxlabs/x";
import { useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { Menu } from "@/components";
import { Export } from "@/export";
import { Group } from "@/group";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Link } from "@/link";
import { Ontology } from "@/ontology";
import { useConfirmDelete } from "@/ontology/hooks";

const useDelete = ({
  selection: { ids },
  state: { getResource },
  removeLayout,
}: Ontology.TreeContextMenuProps): (() => void) => {
  const confirm = useConfirmDelete({ type: "LinePlot" });
  const keys = useMemo(() => ids.map((id) => id.key), [ids]);
  const dispatch = useDispatch();
  const beforeUpdate = useCallback(async () => {
    const ok = await confirm(getResource(ids));
    if (!ok) return false;
    removeLayout(...keys);
    dispatch(LinePlot.remove({ keys }));
    return false;
  }, [confirm, dispatch, keys, removeLayout]);
  const { update } = Core.useDelete({ beforeUpdate });
  return useCallback(() => update(keys), [update, keys]);
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { ids, rootID },
    state: { getResource, shape },
  } = props;
  const handleDelete = useDelete(props);
  const handleLink = Cluster.useCopyLinkToClipboard();
  const handleExport = LinePlot.useExport();
  const group = Group.useCreateFromSelection();
  const firstID = ids[0];
  const isSingle = ids.length === 1;
  const first = getResource(firstID);
  const onSelect = {
    delete: handleDelete,
    rename: () => Text.edit(ontology.idToString(firstID)),
    link: () => handleLink({ name: first.name, ontologyID: firstID }),
    export: () => handleExport(first.id.key),
    group: () => group(props),
  };
  return (
    <PMenu.Menu onChange={onSelect} level="small" gap="small">
      {isSingle && (
        <>
          <Menu.RenameItem />
          <PMenu.Divider />
        </>
      )}
      <Group.MenuItem ids={ids} shape={shape} rootID={rootID} />
      <PMenu.Item itemKey="delete">
        <Icon.Delete />
        Delete
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
  eager: ({ store, id, name }) => store.dispatch(Layout.rename({ key: id.key, name })),
  execute: async ({ client, id, name }) =>
    await client.workspaces.lineplots.rename(id.key, name),
  rollback: ({ store, id }, prevName) =>
    store.dispatch(Layout.rename({ key: id.key, name: prevName })),
};

const handleSelect: Ontology.HandleSelect = ({
  client,
  selection,
  placeLayout,
  handleError,
}) => {
  client.workspaces.lineplots
    .retrieve({ key: selection[0].id.key })
    .then((linePlot) => {
      placeLayout(
        LinePlot.create({ ...linePlot.data, key: linePlot.key, name: linePlot.name }),
      );
    })
    .catch((e) => {
      const names = strings.naturalLanguageJoin(
        selection.map(({ name }) => name),
        "line plot",
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
}): void =>
  handleError(async () => {
    const linePlot = await client.workspaces.lineplots.retrieve({ key });
    placeLayout(
      LinePlot.create({
        ...linePlot.data,
        key: linePlot.key,
        name: linePlot.name,
        location: "mosaic",
        tab: { mosaicKey: nodeKey, location },
      }),
    );
  }, "Failed to load line plot");

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "lineplot",
  icon: <Icon.LinePlot />,
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
