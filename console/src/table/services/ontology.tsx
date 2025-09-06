// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, type Synnax } from "@synnaxlabs/client";
import { Icon, Menu as PMenu, Mosaic, Text, Tree } from "@synnaxlabs/pluto";
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
import { Table } from "@/table";

const useDelete = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const confirm = useConfirmDelete({ type: "Table" });
  return useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    onMutate: async ({
      selection,
      removeLayout,
      state: { nodes, setNodes, getResource },
    }) => {
      if (!(await confirm(getResource(selection.resourceIDs))))
        throw new errors.Canceled();
      const ids = ontology.parseIDs(selection.resourceIDs);
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
      const ids = ontology.parseIDs(selection.resourceIDs);
      await new Promise((resolve) => setTimeout(resolve, 1000));
      await client.workspaces.table.delete(ids.map((id) => id.key));
    },
    onError: (e, { state: { setNodes }, handleError }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      if (errors.Canceled.matches(e)) return;
      handleError(e, "Failed to delete table");
    },
  }).mutate;
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { resourceIDs, rootID },
    state: { getResource, shape },
  } = props;
  const del = useDelete();
  const handleLink = Cluster.useCopyLinkToClipboard();
  const handleExport = Table.useExport();
  const group = Group.useCreateFromSelection();
  const firstID = resourceIDs[0];
  const first = getResource(firstID);
  const onSelect = useAsyncActionMenu({
    delete: () => del(props),
    rename: () => Text.edit(ontology.idToString(firstID)),
    link: () => handleLink({ name: first.name, ontologyID: firstID }),
    export: () => handleExport(first.id.key),
    group: () => group(props),
  });
  const isSingle = resourceIDs.length === 1;
  return (
    <PMenu.Menu onChange={onSelect} level="small" gap="small">
      <Menu.RenameItem />
      <Menu.DeleteItem />
      <Group.MenuItem resourceIDs={resourceIDs} shape={shape} rootID={rootID} />
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
    await client.workspaces.table.rename(id.key, name),
  rollback: ({ id: { key }, name, store }) =>
    store.dispatch(Layout.rename({ key, name })),
};

const loadTable = async (
  client: Synnax,
  id: ontology.ID,
  placeLayout: Layout.Placer,
) => {
  const table = await client.workspaces.table.retrieve(id.key);
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
  id,
  location,
  nodeKey,
  placeLayout,
  handleError,
}) =>
  handleError(async () => {
    const table = await client.workspaces.table.retrieve(id.key);
    placeLayout(
      Table.create({
        name: table.name,
        ...table.data,
        key: id.key,
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
  allowRename: () => true,
  onRename: handleRename,
  onMosaicDrop: handleMosaicDrop,
  TreeContextMenu,
};
