// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, type resource } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Access, Arc } from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Arc as PlatformArc } from "@/platform/arc";
import { ContextMenu } from "@/platform/context-menu";
import { Core } from "@/platform/core";
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
    selection: { ids, parentID },
    state: { getResource },
    openTab,
  } = props;
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const canCreateDirectory = Access.useCreateGranted(arc.directory.TYPE_RESOURCE_ID);
  const createDirectory = PlatformArc.useCreateDirectoryFromSelection();
  const rename = useRename(props);
  const handleDelete = useDelete(props);
  const handleLink = Core.useCopyLinkToClipboard();
  const firstID = ids[0];
  const singleResource = ids.length === 1;
  return (
    <ContextMenu.Menu>
      {canCreateDirectory && hasUpdatePermission && parentID != null && (
        <Menu.Item itemKey="newDirectory" onClick={() => createDirectory(props)}>
          <Icon.Group />
          New directory
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
      <Tree.CopyContextMenuItem {...props} />
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

interface TypeIconProps extends Icon.IconProps {
  arcKey: arc.Key;
}

const TypeIcon = ({ arcKey, ...rest }: TypeIconProps): ReactElement | null => {
  const { data: type } = Arc.useResultType({ key: arcKey });
  if (type == null) return null;
  const C = PlatformArc.TYPE_ICONS[type];
  return <C {...rest} />;
};

const TreeItem = Tree.createItem({
  linkable: true,
  icon: ({ id }) => <TypeIcon arcKey={id.key} />,
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
  icon: "Group",
  useUpdate: Arc.Directory.useDelete,
  convertKey: String,
});

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
  return (
    <ContextMenu.Menu>
      {isSingle && hasCreatePermission && (
        <Menu.Item itemKey="newDirectory" onClick={createDirectory}>
          <Icon.Group />
          New directory
        </Menu.Item>
      )}
      <Menu.Divider />
      {isSingle && hasUpdatePermission && <ContextMenu.RenameItem onClick={rename} />}
      <Menu.Divider />
      <Tree.CopyContextMenuItem {...props} />
      {isSingle && (
        <Link.CopyContextMenuItem
          onClick={() =>
            handleLink({ name: state.getResource(firstID).name, resourceID: firstID })
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

/** The menu for a right-click on empty space in a tree of Arcs. */
export const RootContextMenu = ({
  root,
  state,
}: Tree.RootContextMenuProps): ReactElement => {
  const createArc = PlatformArc.useCreate();
  const createDirectory = PlatformArc.useCreateDirectory({ parent: root, state, root });
  const canCreateArc = Access.useCreateGranted(arc.TYPE_RESOURCE_ID);
  const canCreateDirectory = Access.useCreateGranted(arc.directory.TYPE_RESOURCE_ID);
  return (
    <ContextMenu.Menu>
      {canCreateArc && (
        <Menu.Item itemKey="createArc" onClick={createArc}>
          <Icon.Add />
          Create Arc
        </Menu.Item>
      )}
      {canCreateDirectory && (
        <Menu.Item itemKey="newDirectory" onClick={createDirectory}>
          <Icon.Group />
          New directory
        </Menu.Item>
      )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const ENTRY_TYPES = new Set<resource.Type>(["arc", "arc_directory"]);

const DirectoryItem = Tree.createItem({
  icon: <Icon.Group />,
  folder: true,
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
    await client.arcs.directories.move(destination.id, ...items.map(({ id }) => id));
  },
  haulItems: ({ id }) => [id],
  ContextMenu: DirectoryContextMenu,
});

export const TREE_ITEMS = {
  arc: TreeItem,
  arc_directory: DirectoryItem,
} satisfies Tree.Items;
