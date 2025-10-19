// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, type Synnax, table } from "@synnaxlabs/client";
import { Icon, Mosaic, Table as Core } from "@synnaxlabs/pluto";
import { array, strings } from "@synnaxlabs/x";

import { Cluster } from "@/cluster";
import { ContextMenu } from "@/components";
import { Export } from "@/export";
import { Group } from "@/group";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { Ontology } from "@/ontology";
import { createUseDelete } from "@/ontology/createUseDelete";
import { createUseRename } from "@/ontology/createUseRename";
import { Table } from "@/table";

const useDelete = createUseDelete({
  type: "Table",
  query: Core.useDelete,
  convertKey: String,
  beforeUpdate: async ({ data, removeLayout, store }) => {
    removeLayout(...data);
    store.dispatch(Table.remove({ keys: array.toArray(data) }));
    return data;
  },
});

const useRename = createUseRename({
  query: Core.useRename,
  ontologyID: table.ontologyID,
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
  const handleDelete = useDelete(props);
  const copyLink = Cluster.useCopyLinkToClipboard();
  const exportTable = Table.useExport();
  const handleRename = useRename(props);
  const handleLink = () => copyLink({ name: first.name, ontologyID: firstID });
  const handleExport = () => exportTable(first.id.key);
  const firstID = ids[0];
  const first = getResource(firstID);
  const isSingle = ids.length === 1;
  return (
    <>
      <ContextMenu.RenameItem onClick={handleRename} />
      <ContextMenu.DeleteItem onClick={handleDelete} showBottomDivider />
      <Group.ContextMenuItem {...props} showBottomDivider />
      {isSingle && (
        <>
          <Export.ContextMenuItem onClick={handleExport} />
          <Link.CopyContextMenuItem onClick={handleLink} />
          <Ontology.CopyContextMenuItem {...props} showBottomDivider />
        </>
      )}
      <ContextMenu.ReloadConsoleItem />
    </>
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
