// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log, resource } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Access, Log, Mosaic } from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";

import { ContextMenu } from "@/platform/context-menu";
import { Core } from "@/platform/core";
import { Export } from "@/platform/export";
import { Group } from "@/platform/group";
import { Link } from "@/platform/link";
import { Panel } from "@/platform/panel";
import { Tree } from "@/platform/tree";
import { Session } from "@/session";

const useDelete = Tree.createUseDelete({
  type: "Log",
  useUpdate: Log.useDelete,
  convertKey: String,
  beforeUpdate: async ({ data, store }) => {
    store.dispatch(Session.Log.remove({ keys: array.toArray(data) }));
    return data;
  },
});

const useRename = Tree.createUseRename({
  useUpdate: Log.useRename,
  resourceID: log.resourceID,
  convertKey: String,
});

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    selection: { ids, rootID, parentID },
    state: { getResource, shape },
  } = props;
  const handleDelete = useDelete(props);
  const handleLink = Core.useCopyLinkToClipboard();
  const handleExport = Export.useResource();
  const rename = useRename(props);
  const group = Group.useCreateFromSelection();
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const firstID = ids[0];
  const isSingle = ids.length === 1;
  return (
    <ContextMenu.Menu>
      {hasUpdatePermission && (
        <>
          {isSingle && <ContextMenu.RenameItem onClick={rename} />}
          <Group.ContextMenuItem
            ids={ids}
            shape={shape}
            rootID={rootID}
            parentID={parentID}
            onClick={() => group(props)}
          />
        </>
      )}
      <Menu.Divider />
      {isSingle && (
        <>
          <Export.ContextMenuItem onClick={() => handleExport(firstID)} />
          <Link.CopyContextMenuItem
            onClick={() =>
              handleLink({ name: getResource(firstID).name, resourceID: firstID })
            }
          />
          <Tree.CopyPropertiesContextMenuItem {...props} />
        </>
      )}
      <Menu.Divider />
      {hasDeletePermission && <ContextMenu.DeleteItem onClick={handleDelete} />}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

/** Lists the logs that sit in a group. */
export const GROUP_MEMBER: Group.Member = {
  type: "log",
  children: (client, key) =>
    Tree.createSource(client.logs, { groups: [key] }, Tree.toResource("log")),
};

const TreeItem = Tree.createItem({
  type: "log",
  icon: <Icon.Log />,
  useOnSelect: Panel.useOpenResource,
  haulItems: ({ id }) => [Mosaic.createTabCreateHaulItem(resource.idToString(id))],
  ContextMenu: TreeContextMenu,
});

export const TREE_ITEMS = { log: TreeItem } satisfies Tree.Items;
