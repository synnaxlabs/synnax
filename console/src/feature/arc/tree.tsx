// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, resource } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Access, Arc, Task } from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";

import { Arc as PlatformArc } from "@/platform/arc";
import { ContextMenu } from "@/platform/context-menu";
import { Core } from "@/platform/core";
import { CSS } from "@/platform/css";
import { Framer } from "@/platform/framer";
import { Link } from "@/platform/link";
import { Panel } from "@/platform/panel";
import { Tree } from "@/platform/tree";
import { Session } from "@/session";

const useRename = Tree.createUseRename({
  useUpdate: Arc.useRename,
  resourceID: arc.resourceID,
  convertKey: String,
});

const useDelete = Tree.createUseDelete({
  type: "Arc",
  useUpdate: Arc.useDelete,
  convertKey: String,
  beforeUpdate: async ({ data, store }) => {
    store.dispatch(Session.Arc.remove({ keys: array.toArray(data) }));
    return data;
  },
});

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    selection: { ids },
    state: { getResource },
    openTab,
  } = props;
  const keys = ids.map((id) => id.key);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const canControl = Framer.useCanCommand();
  const captureDeploy = PlatformArc.useCaptureDeploy();
  const { update: runCommand } = Task.useCommand({ afterSuccess: captureDeploy });
  const rename = useRename(props);
  const handleDelete = useDelete(props);
  const handleLink = Core.useCopyLinkToClipboard();
  const firstID = ids[0];
  const singleResource = ids.length === 1;
  const driftedTaskKeys = PlatformArc.useDriftedTaskKeys(keys);
  const redeployTaskKeys = canControl ? driftedTaskKeys : [];
  return (
    <ContextMenu.Menu>
      {redeployTaskKeys.length > 0 && (
        <Menu.Item
          className={CSS.BE("arc", "redeploy-item")}
          itemKey="redeploy"
          onClick={() =>
            runCommand(redeployTaskKeys.map((key) => ({ task: key, type: "start" })))
          }
        >
          <Icon.Refresh />
          Redeploy
        </Menu.Item>
      )}
      <Menu.Divider />
      {hasUpdatePermission && singleResource && (
        <>
          <Menu.Item
            itemKey="edit"
            onClick={() => openTab({ variant: "resource", resource: firstID })}
          >
            <Icon.Edit />
            Edit
          </Menu.Item>
          <Menu.Divider />
          <ContextMenu.RenameItem onClick={rename} />
        </>
      )}
      <Menu.Divider />
      {singleResource && (
        <Link.CopyContextMenuItem
          onClick={() =>
            handleLink({ name: getResource(firstID).name, resourceID: firstID })
          }
        />
      )}
      <Menu.Divider />
      {hasDeletePermission && <ContextMenu.DeleteItem onClick={handleDelete} />}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const TreeItem = Tree.createItem({
  type: "arc",
  icon: ({ data }) => {
    const TypeIcon = PlatformArc.TYPE_ICONS[(data as Pick<arc.Arc, "type">).type];
    return <TypeIcon />;
  },
  hasChildren: false,
  useOnSelect: Panel.useOpenResource,
  ContextMenu: TreeContextMenu,
});

const useRenameDirectory = Tree.createUseRename({
  useUpdate: Arc.Directory.useRename,
  resourceID: arc.directory.resourceID,
  convertKey: String,
});

const useDeleteDirectory = Tree.createUseDelete({
  type: "Directory",
  useUpdate: Arc.Directory.useDelete,
  convertKey: String,
});

// The root directory is the one directory that no directory holds.
const isRootDirectory = ({ data }: Tree.Resource): boolean =>
  data != null && (data as Partial<arc.directory.Directory>).parent == null;

const DirectoryContextMenu: Tree.ContextMenu = (props) => {
  const {
    selection: { ids, rootID },
    state,
  } = props;
  const hasCreatePermission = Access.useCreateGranted(arc.directory.TYPE_RESOURCE_ID);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const firstID = ids[0];
  const createDirectory = PlatformArc.useCreateDirectory({
    parent: firstID,
    state,
    root: rootID,
  });
  const rename = useRenameDirectory(props);
  const handleDelete = useDeleteDirectory(props);
  const handleLink = Core.useCopyLinkToClipboard();
  const isSingle = ids.length === 1;
  const isRoot = isRootDirectory(state.getResource(firstID));
  return (
    <ContextMenu.Menu>
      {isSingle && hasCreatePermission && (
        <Menu.Item itemKey="newDirectory" onClick={createDirectory}>
          <Icon.Group />
          New directory
        </Menu.Item>
      )}
      <Menu.Divider />
      {isSingle && hasUpdatePermission && !isRoot && (
        <ContextMenu.RenameItem onClick={rename} />
      )}
      <Menu.Divider />
      {isSingle && (
        <Link.CopyContextMenuItem
          onClick={() =>
            handleLink({ name: state.getResource(firstID).name, resourceID: firstID })
          }
        />
      )}
      <Menu.Divider />
      {hasDeletePermission && !isRoot && (
        <ContextMenu.DeleteItem onClick={handleDelete} />
      )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const ENTRY_TYPES = new Set<resource.Type>(["arc", "arc_directory"]);

const DirectoryItem = Tree.createItem({
  type: "arc_directory",
  icon: <Icon.Group />,
  children: ({ client, id }) => [
    Tree.createSource(
      client.arcs.directories,
      { parents: [id.key] },
      Tree.toResource("arc_directory"),
    ),
    Tree.createSource(client.arcs, { directories: [id.key] }, Tree.toResource("arc")),
  ],
  canDrop: ({ moved }) => moved.every(({ id }) => ENTRY_TYPES.has(id.type)),
  drop: async ({ client, destination, items }) => {
    // Each item leaves its own directory: a selection can span several.
    const bySource = new Map<string, resource.ID[]>();
    items.forEach(({ id, ancestors: [source] }) => {
      const key = resource.idToString(source);
      bySource.set(key, [...(bySource.get(key) ?? []), id]);
    });
    await Promise.all(
      [...bySource].map(
        async ([source, ids]) =>
          await client.arcs.directories.move(
            resource.parseID(source),
            destination.id,
            ...ids,
          ),
      ),
    );
  },
  haulItems: ({ id }) => [id],
  ContextMenu: DirectoryContextMenu,
});

export const TREE_ITEMS = {
  arc: TreeItem,
  arc_directory: DirectoryItem,
} satisfies Tree.Items;
