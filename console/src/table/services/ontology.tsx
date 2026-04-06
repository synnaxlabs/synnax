// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, type Synnax, table } from "@synnaxlabs/client";
import { Access, Icon, Menu, Mosaic, Table as Base } from "@synnaxlabs/pluto";
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
  query: Base.useDelete,
  convertKey: String,
  beforeUpdate: async ({ data, removeLayout, store }) => {
    removeLayout(...data);
    store.dispatch(Table.remove({ keys: array.toArray(data) }));
    return data;
  },
});

const useRename = createUseRename({
  query: Base.useRename,
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
    selection: { ids, rootID },
    state: { getResource, shape },
  } = props;
  const handleDelete = useDelete(props);
  const handleLink = Cluster.useCopyLinkToClipboard();
  const handleExport = Table.useExport();
  const rename = useRename(props);
  const group = Group.useCreateFromSelection();
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const firstID = ids[0];
  const first = getResource(firstID);
  const isSingle = ids.length === 1;
  return (
    <ContextMenu.Menu>
      {hasUpdatePermission && (
        <>
          <ContextMenu.RenameItem onClick={rename} />
          <Group.ContextMenuItem
            ids={ids}
            shape={shape}
            rootID={rootID}
            onClick={() => group(props)}
          />
        </>
      )}
      {hasDeletePermission && <ContextMenu.DeleteItem onClick={handleDelete} />}
      {(hasUpdatePermission || hasDeletePermission) && <Menu.Divider />}
      {isSingle && (
        <>
          <Export.ContextMenuItem onClick={() => handleExport(first.id.key)} />
          <Link.CopyContextMenuItem
            onClick={() => handleLink({ name: first.name, ontologyID: firstID })}
          />
          <Ontology.CopyPropertiesContextMenuItem {...props} />
          <Menu.Divider />
        </>
      )}
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const loadTable = async (
  client: Synnax,
  { key }: ontology.ID,
  placeLayout: Layout.Placer,
) => {
  const table = await client.tables.retrieve({ key });
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
    const table = await client.tables.retrieve({ key });
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
