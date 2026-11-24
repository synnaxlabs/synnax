// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, group, type impact, resource } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import { Access, Group, Synnax } from "@synnaxlabs/pluto";

import { ContextMenu } from "@/platform/context-menu";
import { Core } from "@/platform/core";
import { Group as PlatformGroup } from "@/platform/group";
import { Impact } from "@/platform/impact";
import { Link } from "@/platform/link";
import { Tree } from "@/platform/tree";

const useRename = Tree.createUseRename({
  useUpdate: Group.useRename,
  resourceID: group.resourceID,
  convertKey: String,
});

type GroupChange = "delete_groups" | "ungroup_groups";

const touchesNothing = ({ entries, kept, hidden }: impact.Plan): boolean =>
  entries.length === 0 &&
  kept === 0 &&
  hidden.restrict + hidden.cascade + hidden.detach === 0;

/**
 * Returns a function that plans a change of the selected groups. A plan that touches
 * nothing else applies at once, and any other plan opens the impact dialog.
 */
const useChange = (): ((
  props: Tree.ContextMenuProps,
  type: GroupChange,
) => Promise<void>) => {
  const client = Synnax.use();
  const open = Impact.useDialog();
  return async ({ selection: { ids }, state: { getResource } }, type) => {
    if (client == null) throw new DisconnectedError();
    const change: impact.Change = { type, keys: ids.map(({ key }) => key) };
    const plan = await client.impact.plan(change);
    if (touchesNothing(plan)) {
      await client.impact.apply(change, { token: plan.token });
      return;
    }
    await open({ change, names: getResource(ids).map(({ name }) => name) });
  };
};

const failureOf = (type: GroupChange, count: number): string => {
  const verb = type === "delete_groups" ? "delete" : "ungroup";
  return `Failed to ${verb} ${count === 1 ? "group" : "groups"}`;
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

  const TreeContextMenu: Tree.ContextMenu = (props) => {
    const {
      selection: { ids, rootID, parentID },
      state,
    } = props;
    const { getResource, getChildren, shape } = state;
    const hasUpdatePermission = Access.useUpdateGranted(ids);
    const hasDeletePermission = Access.useDeleteGranted(ids);
    const handleError = Status.useAsyncErrorHandler();
    const change = useChange();
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
    const type: GroupChange = isDelete ? "delete_groups" : "ungroup_groups";
    const handleChange = () =>
      void handleError(
        async () => await change(props, type),
        failureOf(type, ids.length),
      );
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
          <Menu.Item itemKey="ungroup" onClick={handleChange}>
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
    typeName: "Group",
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
