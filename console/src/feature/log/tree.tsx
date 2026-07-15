// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log, ontology, type Synnax as Client } from "@synnaxlabs/client";
import { Access, Icon, Log, Menu, Mosaic, Status, Synnax } from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { useCallback } from "react";

import { useExport } from "@/feature/log/export";
import { Cluster } from "@/platform/cluster";
import { ContextMenu } from "@/platform/context-menu";
import { Export } from "@/platform/export";
import { Group } from "@/platform/group";
import { Layout } from "@/platform/layout";
import { Link } from "@/platform/link";
import { Log as PlatformLog } from "@/platform/log";
import { Tree } from "@/platform/tree";
import { Session } from "@/session";

const useDelete = Tree.createUseDelete({
  type: "Log",
  query: Log.useDelete,
  convertKey: String,
  beforeUpdate: async ({ data, removeLayout, store }) => {
    removeLayout(...data);
    store.dispatch(Session.Log.remove({ keys: array.toArray(data) }));
    return data;
  },
});

const useRename = Tree.createUseRename({
  query: Log.useRename,
  ontologyID: log.ontologyID,
  convertKey: String,
  beforeUpdate: async ({ data, rollbacks, store, oldName }) => {
    const { key, name } = data;
    store.dispatch(Session.Layout.rename({ key, name }));
    rollbacks.push(() => store.dispatch(Session.Layout.rename({ key, name: oldName })));
    return { ...data, name };
  },
});

const TreeContextMenu: Tree.ContextMenu = (props) => {
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
          <Tree.CopyPropertiesContextMenuItem {...props} />
          <Menu.Divider />
        </>
      )}
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const loadLog = async (
  client: Client,
  { key }: ontology.ID,
  placeLayout: Layout.Placer,
) => {
  const l = await client.logs.retrieve({ key });
  placeLayout(PlatformLog.create({ key: l.key, name: l.name }));
};

const useOnSelect = (): ((resource: ontology.Resource) => void) => {
  const client = Synnax.use();
  const placeLayout = Layout.usePlacer();
  const handleError = Status.useErrorHandler();
  return useCallback(
    (resource) => {
      if (client == null) return;
      loadLog(client, resource.id, placeLayout).catch((e: unknown) =>
        handleError(e, `Failed to select ${resource.name}`),
      );
    },
    [client, placeLayout, handleError],
  );
};

const TreeItem = Tree.createItem({
  type: "log",
  icon: <Icon.Log />,
  hasChildren: false,
  useOnSelect,
  haulItems: ({ id }) => [Mosaic.createTabCreateHaulItem(ontology.idToString(id))],
  ContextMenu: TreeContextMenu,
});

export const TREE_ITEMS = { log: TreeItem } satisfies Tree.Items;
