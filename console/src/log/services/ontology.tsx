// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log, ontology, type Synnax } from "@synnaxlabs/client";
import {
  ContextMenu as PContextMenu,
  Icon,
  Log as Core,
  Mosaic,
} from "@synnaxlabs/pluto";
import { array, strings } from "@synnaxlabs/x";

import { Cluster } from "@/cluster";
import { ContextMenu } from "@/components";
import { Export } from "@/export";
import { Group } from "@/group";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { Log } from "@/log";
import { Ontology } from "@/ontology";
import { createUseDelete } from "@/ontology/createUseDelete";
import { createUseRename } from "@/ontology/createUseRename";

const useDelete = createUseDelete({
  type: "Log",
  query: Core.useDelete,
  convertKey: String,
  beforeUpdate: async ({ data, removeLayout, store }) => {
    removeLayout(...data);
    store.dispatch(Log.remove({ keys: array.toArray(data) }));
    return data;
  },
});

const useRename = createUseRename({
  query: Core.useRename,
  ontologyID: log.ontologyID,
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
  const exportLog = Log.useExport();
  const handleExport = () => exportLog(ids[0].key);
  const handleRename = useRename(props);
  const firstID = ids[0];
  const firstResource = getResource(firstID);
  const handleLink = () => copyLink({ name: firstResource.name, ontologyID: firstID });
  const isSingle = ids.length === 1;
  return (
    <>
      <ContextMenu.RenameItem onClick={handleRename} />
      <ContextMenu.DeleteItem onClick={handleDelete} />
      <Group.ContextMenuItem {...props} />
      <PContextMenu.Divider />
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

const loadLog = async (
  client: Synnax,
  { key }: ontology.ID,
  placeLayout: Layout.Placer,
) => {
  const log = await client.workspaces.logs.retrieve({ key });
  placeLayout(Log.create({ ...(log.data as Log.State), key: log.key, name: log.name }));
};

const handleSelect: Ontology.HandleSelect = ({
  client,
  selection,
  placeLayout,
  handleError,
}) => {
  loadLog(client, selection[0].id, placeLayout).catch((e) => {
    const names = strings.naturalLanguageJoin(
      selection.map(({ name }) => name),
      "log",
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
    const log = await client.workspaces.logs.retrieve({ key });
    placeLayout(
      Log.create({
        name: log.name,
        ...log.data,
        key,
        location: "mosaic",
        tab: { mosaicKey: nodeKey, location },
      }),
    );
  }, "Failed to load log");

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "log",
  icon: <Icon.Log />,
  hasChildren: false,
  onSelect: handleSelect,
  haulItems: ({ id }) => [
    { type: Mosaic.HAUL_CREATE_TYPE, key: ontology.idToString(id) },
  ],
  onMosaicDrop: handleMosaicDrop,
  TreeContextMenu,
};
