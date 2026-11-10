// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, resource } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Access, Flux, Group } from "@synnaxlabs/pluto";

import { ContextMenu } from "@/platform/context-menu";
import { Core } from "@/platform/core";
import { Group as PlatformGroup } from "@/platform/group";
import { Link } from "@/platform/link";
import { Tree } from "@/platform/tree";

const useRename = Tree.createUseRename({
  useUpdate: Group.useRename,
  resourceID: group.resourceID,
  convertKey: String,
});

const parentOf = ({ parentID }: Tree.ContextMenuProps["selection"]): resource.ID => {
  if (parentID == null)
    throw new Error("Cannot ungroup groups that sit under more than one parent");
  return parentID;
};

const retrieve: Tree.Retrieve = async (c, key) => await c.groups.retrieve({ key });

/**
 * Returns the tree items of groups. A group lists its child groups and the members of
 * its scope.
 */
export const createTreeItems = (
  members: PlatformGroup.Member[],
): { group: Tree.Item } => {
  const membersOf = ({ type }: resource.ID): PlatformGroup.Member[] => {
    if (!PlatformGroup.isScope(type)) throw new Error(`no tree holds ${type} groups`);
    return members.filter((m) => group.MEMBER_RULES[m.type].scope === type);
  };

  const children: Tree.Children = ({ client, id, root }) => [
    Tree.createSource(client.groups, { parents: [id] }, Tree.toResource("group")),
    ...membersOf(root).map((m) => m.children(client, id.key)),
  ];

  const retrieveChildren = async (
    props: Tree.ChildrenProps,
  ): Promise<resource.ID[]> => {
    const resources = await Promise.all(
      children(props).map(async (source) => await source.retrieveAll()),
    );
    return resources.flat().map(({ id }) => id);
  };

  // Ungroup reads children from the Core, since the tree holds only the pages it shows.
  const ungroup = Flux.createUpdate<Tree.ContextMenuProps>({
    name: "resources",
    verbs: {
      present: "ungroup",
      past: "ungrouped",
      participle: "ungrouping",
    },
    update: async ({ client, data }) => {
      const { selection } = data;
      const { ids, rootID } = selection;
      const parentID = parentOf(selection);
      const selected = new Set(resource.idToString(ids));
      const childIDs = (
        await Promise.all(
          ids.map(async (id) => await retrieveChildren({ client, id, root: rootID })),
        )
      )
        .flat()
        .filter((id) => !selected.has(resource.idToString(id)));
      const parent = resource.idsEqual(parentID, rootID) ? undefined : parentID;
      await client.batch.send([
        ...client.groups.moveRequests(childIDs, parent),
        client.groups.deleteRequest(ids.map(({ key }) => key)),
      ]);
      return data;
    },
  });

  const TreeContextMenu: Tree.ContextMenu = (props) => {
    const {
      selection: { ids, rootID, parentID },
      state,
    } = props;
    const { getResource, getChildren, shape } = state;
    const hasUpdatePermission = Access.useUpdateGranted(ids);
    const hasDeletePermission = Access.useDeleteGranted(ids);
    const { update: handleUngroup } = ungroup.useUpdate();
    const createEmptyGroup = PlatformGroup.useCreateEmpty({
      parent: ids[0],
      state,
      root: rootID,
    });
    const createFromSelection = PlatformGroup.useCreateFromSelection();
    const handleLink = Core.useCopyLinkToClipboard();
    const firstID = ids[0];
    const rename = useRename(props);
    const isSingle = ids.length === 1;
    // A closed group has unknown children, so it shows "Ungroup".
    const isDelete = ids.every(
      (id) => getChildren(resource.idToString(id))?.length === 0,
    );
    const ungroupIcon = isDelete ? <Icon.Delete /> : <Icon.Group />;
    return (
      <ContextMenu.Menu>
        {isSingle && hasUpdatePermission && <ContextMenu.RenameItem onClick={rename} />}
        <Menu.Divider />
        {isSingle && hasUpdatePermission && (
          <Menu.Item itemKey="newGroup" onClick={createEmptyGroup}>
            <Icon.Group />
            New group
          </Menu.Item>
        )}
        {hasUpdatePermission && (
          <PlatformGroup.ContextMenuItem
            ids={ids}
            shape={shape}
            rootID={rootID}
            parentID={parentID}
            onClick={() => createFromSelection(props)}
          />
        )}
        <Menu.Divider />
        <Tree.CopyContextMenuItem {...props} />
        {isSingle && (
          <>
            <Link.CopyContextMenuItem
              onClick={() =>
                handleLink({ name: getResource(firstID).name, resourceID: firstID })
              }
            />
            <Tree.CopyPropertiesContextMenuItem {...props} retrieve={retrieve} />
          </>
        )}
        <Menu.Divider />
        {hasDeletePermission && parentID != null && (
          <Menu.Item itemKey="ungroup" onClick={() => handleUngroup(props)}>
            {ungroupIcon}
            {isDelete ? "Delete" : "Ungroup"}
          </Menu.Item>
        )}
        <Menu.Divider />
        <ContextMenu.ReloadConsoleItem />
      </ContextMenu.Menu>
    );
  };

  const TreeItem = Tree.createItem({
    icon: <Icon.Group />,
    children,
    folder: true,
    canDrop: (props) => {
      const types = new Set<resource.Type>([
        "group",
        ...membersOf(props.root).map(({ type }) => type),
      ]);
      return (
        props.moved.every(({ id }) => types.has(id.type)) &&
        group.canMove({ ...props, destination: props.destination.id })
      );
    },
    drop: async ({ client, destination, items }) => {
      await client.batch.send(
        client.groups.moveRequests(
          items.map(({ id }) => id),
          destination.id,
        ),
      );
    },
    haulItems: ({ id }) => [id],
    ContextMenu: TreeContextMenu,
  });

  return { group: TreeItem };
};
