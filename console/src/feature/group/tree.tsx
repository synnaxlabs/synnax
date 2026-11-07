// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, ontology, resource } from "@synnaxlabs/client";
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

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    selection: { ids, rootID, parentID },
    state,
  } = props;
  const { getResource, nodes, shape } = state;
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const ungroup = useUngroupSelection();
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
  const isZeroDepth =
    PTree.getDepth(resource.idToString(firstID), shape) === 0 &&
    resource.idsEqual(rootID, ontology.ROOT_ID);
  const isDelete = ids.every((id) => {
    const node = PTree.findNode({ tree: nodes, key: resource.idToString(id) });
    return node?.children == null || node?.children.length === 0;
  });
  const ungroupIcon = isDelete ? <Icon.Delete /> : <Icon.Group />;
  return (
    <ContextMenu.Menu>
      {isSingle && hasUpdatePermission && !isZeroDepth && (
        <ContextMenu.RenameItem onClick={rename} />
      )}
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
      {hasDeletePermission && !isZeroDepth && parentID != null && (
        <Menu.Item itemKey="ungroup" onClick={() => ungroup.update(props)}>
          {ungroupIcon}
          {isDelete ? "Delete" : "Ungroup"}
        </Menu.Item>
      )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const parentOf = ({ parentID }: Tree.ContextMenuProps["selection"]): resource.ID => {
  if (parentID == null)
    throw new Error("Cannot ungroup groups that sit under more than one parent");
  return parentID;
};

interface UngroupParams extends Tree.ContextMenuProps {
  prevNodes?: PTree.Node<string>[];
}

const baseUngroup = Flux.createUpdate<UngroupParams>({
  name: "resources",
  verbs: {
    present: "ungroup",
    past: "ungrouped",
    participle: "ungrouping",
  },
  update: async ({ client, data: args }) => {
    const { selection, prevNodes } = args;
    if (prevNodes == null) return args;
    const parentID = parentOf(selection);
    const resourceIDStrings = new Set(
      selection.ids.map((id) => resource.idToString(id)),
    );
    for (const id of selection.ids) {
      // The tree can hold only some of the children: a search shows only the matches,
      // and a folded group loads none.
      const children = await client.ontology.children.retrieve({ ids: id });
      const childIDs = children
        .map((child) => child.id)
        .filter((child) => !resourceIDStrings.has(resource.idToString(child)));
      await client.ontology.moveChildren(id, parentID, ...childIDs);
    }
    await client.groups.delete(selection.ids.map((id) => id.key));
    return args;
  },
});

const beforeUngroup = async ({ data }: Flux.BeforeUpdateParams<UngroupParams>) => {
  const {
    selection,
    state: { shape, nodes, setNodes },
  } = data;
  const parentKey = resource.idToString(parentOf(selection));
  // Flat search results do not show the real parent, so the children move to the top.
  const destination =
    PTree.findNode({ tree: nodes, key: parentKey }) == null ? null : parentKey;
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
          destination,
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

const useUngroupSelection = () =>
  baseUngroup.useUpdate({
    beforeUpdate: beforeUngroup,
    afterFailure: afterUngroupFailure,
  });

const TreeItem = Tree.createItem({
  type: "group",
  icon: <Icon.Group />,
  canDrop: () => true,
  haulItems: ({ id }) => [id],
  ContextMenu: TreeContextMenu,
});

export const TREE_ITEMS = { group: TreeItem } satisfies Tree.Items;
