// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log, ontology, type Synnax } from "@synnaxlabs/client";
import { Access, Icon, Log, Menu, Mosaic } from "@synnaxlabs/pluto";
import { array, strings } from "@synnaxlabs/x";

import { useExport } from "@/feature/log/export";
import { Cluster } from "@/platform/cluster";
import { ContextMenu } from "@/platform/context-menu";
import { Export } from "@/platform/export";
import { Group } from "@/platform/group";
import { Link } from "@/platform/link";
import { Ontology } from "@/platform/ontology";
import { type Panel } from "@/platform/panel";
import { Session } from "@/session";

const useDelete = Ontology.createUseDelete({
  type: "Log",
  query: Log.useDelete,
  convertKey: String,
  beforeUpdate: async ({ data, store }) => {
    store.dispatch(Session.Log.remove({ keys: array.toArray(data) }));
    return data;
  },
});

const useRename = Ontology.createUseRename({
  query: Log.useRename,
  ontologyID: log.ontologyID,
  convertKey: String,
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
  const firstResource = getResource(firstID);
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
          <Export.ContextMenuItem onClick={() => handleExport(ids[0].key)} />
          <Link.CopyContextMenuItem
            onClick={() => handleLink({ name: firstResource.name, ontologyID: ids[0] })}
          />
          <Ontology.CopyPropertiesContextMenuItem {...props} />
          <Menu.Divider />
        </>
      )}
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const loadLog = async (
  client: Synnax,
  { key }: ontology.ID,
  openTab: Panel.OpenTab,
) => {
  const l = await client.logs.retrieve({ key });
  openTab({ variant: "resource", resource: log.ontologyID(l.key) });
};

const handleSelect: Ontology.HandleSelect = ({
  client,
  selection,
  openTab,
  handleError,
}) => {
  loadLog(client, selection[0].id, openTab).catch((e: unknown) => {
    const names = strings.naturalLanguageJoin(
      selection.map(({ name }) => name),
      "log",
    );
    handleError(e, `Failed to select ${names}`);
  });
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "log",
  icon: <Icon.Log />,
  hasChildren: false,
  onSelect: handleSelect,
  haulItems: ({ id }) => [Mosaic.createTabCreateHaulItem(ontology.idToString(id))],
  TreeContextMenu,
};
