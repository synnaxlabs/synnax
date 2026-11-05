// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, type ontology, resource } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Tree as PTree } from "@synnaxlabs/lyra/tree";
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

const parseScope = (res?: ontology.Resource): PlatformGroup.Scope => {
  const { scope } = group.groupZ.parse(res?.data);
  if (!PlatformGroup.isScope(scope)) throw new Error(`no tree holds ${scope} groups`);
  return scope;
};

interface UngroupParams extends Tree.ContextMenuProps {
  prevNodes?: PTree.Node<string>[];
}

const beforeUngroup = async ({ data }: Flux.BeforeUpdateParams<UngroupParams>) => {
  const {
    selection,
    state: { shape, nodes, setNodes },
  } = data;
  // Sort the groups by depth that way deeper nested groups are ungrouped first.
  const byDepth = PTree.compareDepth(shape);
  selection.ids.sort((a, b) => byDepth(resource.idToString(a), resource.idToString(b)));
  const prevNodes = PTree.deepCopy(nodes);
  const nextNodes = [
    ...selection.ids.reduce(
      (acc, id) => {
        const key = resource.idToString(id);
        const children = PTree.findNode({ tree: nodes, key })?.children ?? [];
        acc = PTree.moveNode({
          tree: acc,
          destination: resource.idsEqual(selection.parentID, selection.rootID)
            ? null
            : resource.idToString(selection.parentID),
          keys: children.map((c) => c.key),
        });
        acc = PTree.removeNode({ tree: acc, keys: key });
        return [...acc];
      },
      [...nodes],
    ),
  ];
  setNodes(nextNodes);
  return { ...data, prevNodes };
};

const afterUngroupFailure = async ({
  data: {
    prevNodes,
    state: { setNodes },
  },
}: Flux.AfterFailureParams<UngroupParams>) => {
  if (prevNodes != null) setNodes(prevNodes);
};

/**
 * Returns the tree items of groups. A group lists its child groups and the members of
 * its scope.
 */
export const createTreeItems = (
  members: PlatformGroup.Members,
): { group: Tree.Item } => {
  const children: Tree.Children = ({ client, id, resource: res }) => [
    Tree.createSource(client.groups, { parents: [id] }, Tree.toResource("group")),
    ...members[parseScope(res)].map((m) => m.children(client, id.key)),
  ];

  const retrieveChildren = async (
    props: Tree.ChildrenProps,
  ): Promise<resource.ID[]> => {
    const answers = await Promise.all(
      children(props).map(async (source) => await source.retrieve()),
    );
    return answers.flat().map(({ id }) => id);
  };

  // Ungroup reads children from the Core, since the tree holds only loaded ones.
  const ungroup = Flux.createUpdate<UngroupParams>({
    name: "resources",
    verbs: {
      present: "ungroup",
      past: "ungrouped",
      participle: "ungrouping",
    },
    update: async ({ client, data }) => {
      const {
        selection: { ids, parentID, rootID },
        state: { getResource },
      } = data;
      const selected = new Set(resource.idToString(ids));
      const childIDs = (
        await Promise.all(
          ids.map(
            async (id) =>
              await retrieveChildren({ client, id, resource: getResource(id) }),
          ),
        )
      )
        .flat()
        .filter((id) => !selected.has(resource.idToString(id)));
      const parent = resource.idsEqual(parentID, rootID) ? undefined : parentID;
      await client.batch.send([
        ...PlatformGroup.moveRequests(client, childIDs, parent),
        client.groups.deleteRequest(ids.map(({ key }) => key)),
      ]);
      return data;
    },
  });

  const useUngroup = () =>
    ungroup.useUpdate({
      beforeUpdate: beforeUngroup,
      afterFailure: afterUngroupFailure,
    });

  const TreeContextMenu: Tree.ContextMenu = (props) => {
    const {
      selection: { ids, rootID },
      state,
    } = props;
    const { getResource, nodes, shape } = state;
    const hasUpdatePermission = Access.useUpdateGranted(ids);
    const hasDeletePermission = Access.useDeleteGranted(ids);
    const { update: handleUngroup } = useUngroup();
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
    const isDelete = ids.every((id) => {
      const node = PTree.findNode({ tree: nodes, key: resource.idToString(id) });
      return node?.children == null || node?.children.length === 0;
    });
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
            onClick={() => createFromSelection(props)}
          />
        )}
        <Menu.Divider />
        {isSingle && (
          <>
            <Link.CopyContextMenuItem
              onClick={() =>
                handleLink({ name: getResource(firstID).name, resourceID: firstID })
              }
            />
            <Tree.CopyPropertiesContextMenuItem {...props} />
          </>
        )}
        <Menu.Divider />
        {hasDeletePermission && (
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
    type: "group",
    icon: <Icon.Group />,
    children,
    canDrop: ({ items, destination }) => {
      const types = new Set<resource.Type>([
        "group",
        ...members[parseScope(destination)].map(({ type }) => type),
      ]);
      return PTree.filterHaulItems(items).every(({ key }) =>
        types.has(resource.parseID(key).type),
      );
    },
    drop: async ({ client, destination, items }) => {
      await client.batch.send(
        PlatformGroup.moveRequests(
          client,
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
