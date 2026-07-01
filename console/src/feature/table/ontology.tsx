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

import { Group } from "@/feature/group";
import { Link } from "@/feature/link";
import { Ontology } from "@/feature/ontology";
import { useExport } from "@/feature/table/export";
import { Cluster } from "@/platform/cluster";
import { ContextMenu } from "@/platform/context-menu";
import { Export } from "@/platform/export";
import { Table } from "@/platform/table";
import { Session } from "@/session";

const useDelete = Ontology.createUseDelete({
  type: "Table",
  query: Base.useDelete,
  convertKey: String,
  beforeUpdate: async ({ data, removeLayout, store }) => {
    removeLayout(...data);
    store.dispatch(Session.Table.remove({ keys: array.toArray(data) }));
    return data;
  },
});

const useRename = Ontology.createUseRename({
  query: Base.useRename,
  ontologyID: table.ontologyID,
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
  const handleDelete = useDelete(props);
  const handleLink = Cluster.useCopyLinkToClipboard();
  const handleExport = useExport();
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
  placeLayout: Session.Layout.Placer,
) => {
  const t = await client.tables.retrieve({ key });
  placeLayout(Table.create({ key: t.key, name: t.name }));
};

const handleSelect: Ontology.HandleSelect = ({
  client,
  selection,
  placeLayout,
  handleError,
}) => {
  loadTable(client, selection[0].id, placeLayout).catch((e: unknown) => {
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
    const t = await client.tables.retrieve({ key });
    placeLayout(
      Table.create({
        key: t.key,
        name: t.name,
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
  haulItems: ({ id }) => [Mosaic.createTabCreateHaulItem(ontology.idToString(id))],
  onMosaicDrop: handleMosaicDrop,
  TreeContextMenu,
};
