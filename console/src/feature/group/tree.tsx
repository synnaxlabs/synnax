// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, ontology } from "@synnaxlabs/client";
import { Access, Flux, Group, Icon, Menu, Tree as PTree } from "@synnaxlabs/pluto";

import { Cluster } from "@/platform/cluster";
import { ContextMenu } from "@/platform/context-menu";
import { Group as PlatformGroup } from "@/platform/group";
import { Link } from "@/platform/link";
import { Tree } from "@/platform/tree";

const useRename = Tree.createUseRename({
  query: Group.useRename,
  ontologyID: group.ontologyID,
  convertKey: String,
});

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    selection: { ids, rootID },
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
  const handleLink = Cluster.useCopyLinkToClipboard();
  const firstID = ids[0];
  const firstResource = getResource(firstID);
  const rename = useRename(props);
  const isSingle = ids.length === 1;
  const isZeroDepth =
    PTree.getDepth(ontology.idToString(firstID), shape) === 0 &&
    ontology.idsEqual(rootID, ontology.ROOT_ID);
  const isDelete = ids.every((id) => {
    const node = PTree.findNode({ tree: nodes, key: ontology.idToString(id) });
    return node?.children == null || node?.children.length === 0;
  });
  const ungroupIcon = isDelete ? <Icon.Delete /> : <Icon.Group />;
  return (
    <ContextMenu.Menu>
      {isSingle && (
        <>
          {hasUpdatePermission && !isZeroDepth && (
            <>
              <ContextMenu.RenameItem onClick={rename} />
              <Menu.Divider />
            </>
          )}
          {hasUpdatePermission && (
            <Menu.Item itemKey="newGroup" onClick={createEmptyGroup}>
              <Icon.Group />
              New group
            </Menu.Item>
          )}
        </>
      )}
      {hasUpdatePermission && (
        <PlatformGroup.ContextMenuItem
          ids={ids}
          shape={shape}
          rootID={rootID}
          onClick={() => createFromSelection(props)}
        />
      )}
      {hasDeletePermission && !isZeroDepth && (
        <>
          <Menu.Item itemKey="ungroup" onClick={() => ungroup.update(props)}>
            {ungroupIcon}
            {isDelete ? "Delete" : "Ungroup"}
          </Menu.Item>
          <Menu.Divider />
        </>
      )}
      {isSingle && (
        <>
          <Tree.CopyPropertiesContextMenuItem {...props} />
          <Link.CopyContextMenuItem
            onClick={() =>
              handleLink({ name: firstResource.name, ontologyID: firstID })
            }
          />
          <Menu.Divider />
        </>
      )}
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

interface UngroupParams extends Tree.ContextMenuProps {
  prevNodes?: PTree.Node<string>[];
}

const baseUngroup = Flux.createUpdate<UngroupParams>({
  name: Group.RESOURCE_NAME,
  verbs: {
    present: "ungroup",
    past: "ungrouped",
    participle: "ungrouping",
  },
  update: async ({ client, data: args }) => {
    const { selection, prevNodes } = args;
    if (prevNodes == null) return args;
    const resourceIDStrings = new Set(
      selection.ids.map((id) => ontology.idToString(id)),
    );
    for (const id of selection.ids) {
      const children =
        PTree.findNode({ tree: prevNodes, key: ontology.idToString(id) })?.children ??
        [];
      const childKeys = ontology.parseIDs(
        children.map(({ key }) => key).filter((k) => !resourceIDStrings.has(k)),
      );
      await client.ontology.moveChildren(id, selection.parentID, ...childKeys);
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
  // Sort the groups by depth that way deeper nested groups are ungrouped first.
  selection.ids.sort(
    (a, b) =>
      PTree.getDepth(ontology.idToString(a), shape) -
      PTree.getDepth(ontology.idToString(b), shape),
  );
  const prevNodes = PTree.deepCopy(nodes);
  const nextNodes = [
    ...selection.ids.reduce(
      (acc, id) => {
        const key = ontology.idToString(id);
        const children = PTree.findNode({ tree: nodes, key })?.children ?? [];
        acc = PTree.moveNode({
          tree: acc,
          destination: ontology.idsEqual(selection.parentID, selection.rootID)
            ? null
            : ontology.idToString(selection.parentID),
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
  // This haul item allows the group to be dragged between nodes in the tree.
  haulItems: ({ id }) => [id],
  ContextMenu: TreeContextMenu,
});

export const TREE_ITEMS = { group: TreeItem } satisfies Tree.Items;
