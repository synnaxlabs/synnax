// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, ontology } from "@synnaxlabs/client";
import { Access, Icon, LinePlot as Base, Menu, Mosaic } from "@synnaxlabs/pluto";
import { array, strings } from "@synnaxlabs/x";

import { Cluster } from "@/cluster";
import { ContextMenu } from "@/components";
import { Export } from "@/export";
import { Group } from "@/group";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Link } from "@/link";
import { Ontology } from "@/ontology";
import { createUseDelete } from "@/ontology/createUseDelete";
import { createUseRename } from "@/ontology/createUseRename";

const useDelete = createUseDelete({
  type: "Line Plot",
  icon: "LinePlot",
  query: Base.useDelete,
  convertKey: String,
  beforeUpdate: async ({ data, removeLayout, store }) => {
    removeLayout(...data);
    store.dispatch(LinePlot.remove({ keys: array.toArray(data) }));
    return data;
  },
});

const useRename = createUseRename({
  query: Base.useRename,
  ontologyID: lineplot.ontologyID,
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
  const handleExport = LinePlot.useExport();
  const rename = useRename(props);
  const group = Group.useCreateFromSelection();
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const firstID = ids[0];
  const isSingle = ids.length === 1;
  const first = getResource(firstID);
  return (
    <ContextMenu.Menu>
      {hasUpdatePermission && (
        <>
          {isSingle && (
            <>
              <ContextMenu.RenameItem onClick={rename} />
              <Menu.Divider />
            </>
          )}
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

const handleSelect: Ontology.HandleSelect = ({
  client,
  selection,
  placeLayout,
  handleError,
}) => {
  const names = strings.naturalLanguageJoin(
    selection.map(({ name }) => name),
    "line plot",
  );
  handleError(async () => {
    const linePlot = await client.lineplots.retrieve({
      key: selection[0].id.key,
    });
    placeLayout(
      LinePlot.create({ ...linePlot.data, key: linePlot.key, name: linePlot.name }),
    );
  }, `Failed to select ${names}`);
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
    const linePlot = await client.lineplots.retrieve({ key });
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
  onMosaicDrop: handleMosaicDrop,
  TreeContextMenu,
};
