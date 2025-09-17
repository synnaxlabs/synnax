// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, ontology } from "@synnaxlabs/client";
import { Flux, Group, Icon, Menu as PMenu, Text, Tree } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Cluster } from "@/cluster";
import { Menu } from "@/components/menu";
import { MenuItem } from "@/group/MenuItem";
import { useCreateFromSelection } from "@/group/useCreateFromSelection";
import { useAsyncActionMenu } from "@/hooks/useAsyncAction";
import { Link } from "@/link";
import { Ontology } from "@/ontology";

const useRename = ({
  selection: {
    ids: [firstID],
  },
}: Ontology.TreeContextMenuProps) => {
  const { update } = Group.useRename({
    beforeUpdate: async ({ value }) => {
      const { key } = value;
      const [name, renamed] = await Text.asyncEdit(
        ontology.idToString(group.ontologyID(key)),
      );
      if (!renamed) return false;
      return { key, name };
    },
  });
  return useCallback(() => update({ key: firstID.key, name: "" }), [firstID]);
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { ids, rootID },
    state: { getResource, nodes, shape },
  } = props;
  const ungroup = useUngroupSelection();
  const createEmptyGroup = useCreateEmpty(props);
  const createFromSelection = useCreateFromSelection();
  const handleLink = Cluster.useCopyLinkToClipboard();
  const firstID = ids[0];
  const firstResource = getResource(firstID);
  const rename = useRename(props);
  const isSingle = ids.length === 1;
  const isZeroDepth =
    Tree.getDepth(ontology.idToString(firstID), shape) === 0 &&
    ontology.idsEqual(rootID, ontology.ROOT_ID);
  const onSelect = useAsyncActionMenu({
    ungroup: () => ungroup.update(props),
    rename,
    newGroup: createEmptyGroup,
    group: () => createFromSelection(props),
    link: () => handleLink({ name: firstResource.name, ontologyID: firstID }),
  });
  const isDelete = ids.every((id) => {
    const node = Tree.findNode({ tree: nodes, key: ontology.idToString(id) });
    return node?.children == null || node?.children.length === 0;
  });
  const ungroupIcon = isDelete ? <Icon.Delete /> : <Icon.Group />;
  return (
    <PMenu.Menu onChange={onSelect} level="small" gap="small">
      {isSingle && (
        <>
          {!isZeroDepth && (
            <>
              <Menu.RenameItem />
              <PMenu.Divider />
            </>
          )}
          <PMenu.Item itemKey="newGroup">
            <Icon.Group />
            New Group
          </PMenu.Item>
        </>
      )}
      <MenuItem ids={ids} shape={shape} rootID={rootID} />
      {!isZeroDepth && (
        <>
          <PMenu.Item itemKey="ungroup">
            {ungroupIcon}
            {isDelete ? "Delete" : "Ungroup"}
          </PMenu.Item>
          <PMenu.Divider />
        </>
      )}
      {isSingle && (
        <>
          <Link.CopyMenuItem />
          <PMenu.Divider />
        </>
      )}
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

interface UseUngroupArgs extends Ontology.TreeContextMenuProps {
  prevNodes?: Tree.Node<string>[];
}

const useUngroupSelection = () =>
  Flux.createUpdate<UseUngroupArgs>({
    name: "Group",
    update: async ({ client, value: args }) => {
      const { selection, prevNodes } = args;
      if (selection.parentID == null || prevNodes == null) return args;
      const resourceIDStrings = new Set(
        selection.ids.map((id) => ontology.idToString(id)),
      );
      for (const id of selection.ids) {
        const children =
          Tree.findNode({ tree: prevNodes, key: ontology.idToString(id) })?.children ??
          [];
        const parentID = selection.parentID;
        const childKeys = ontology.parseIDs(
          children.map(({ key }) => key).filter((k) => !resourceIDStrings.has(k)),
        );
        await client.ontology.moveChildren(id, parentID, ...childKeys);
      }
      await client.ontology.groups.delete(...selection.ids.map((id) => id.key));
      return args;
    },
  }).useUpdate({
    beforeUpdate: async ({ value: args }: Flux.BeforeUpdateArgs<UseUngroupArgs>) => {
      const {
        selection,
        state: { shape, nodes, setNodes },
      } = args;
      if (selection.parentID == null) return false;
      // Sort the groups by depth that way deeper nested groups are ungrouped first.
      selection.ids.sort(
        (a, b) =>
          Tree.getDepth(ontology.idToString(a), shape) -
          Tree.getDepth(ontology.idToString(b), shape),
      );
      const prevNodes = Tree.deepCopy(nodes);
      const nextNodes = [
        ...selection.ids.reduce(
          (acc, id) => {
            const key = ontology.idToString(id);
            const children = Tree.findNode({ tree: nodes, key })?.children ?? [];
            acc = Tree.moveNode({
              tree: acc,
              destination: ontology.idsEqual(selection.parentID, selection.rootID)
                ? null
                : ontology.idToString(selection.parentID),
              keys: children.map((c) => c.key),
            });
            acc = Tree.removeNode({ tree: acc, keys: key });
            return [...acc];
          },
          [...nodes],
        ),
      ];
      setNodes(nextNodes);
      return { ...args, prevNodes };
    },
    afterFailure: async ({
      status,
      value: {
        addStatus,
        prevNodes,
        state: { setNodes },
      },
    }: Flux.AfterFailureArgs<UseUngroupArgs>) => {
      addStatus(status);
      if (prevNodes != null) setNodes(prevNodes);
    },
  });

const useCreateEmpty = (props: Ontology.TreeContextMenuProps) => {
  const { update } = Group.useCreate({
    beforeUpdate: useCallback(
      async ({ value, rollbacks }: Flux.BeforeUpdateArgs<Group.CreateValue>) => {
        const {
          state: { nodes: tree, setNodes, expand, setResource },
        } = props;
        const newID = group.ontologyID(uuid.create());
        const newIDString = ontology.idToString(newID);
        const res: ontology.Resource = { key: newIDString, id: newID, name: "" };
        const node: Tree.Node<string> = { key: newIDString, children: [] };
        setResource(res);
        const destination = ontology.idToString(value.parent);
        expand(destination);
        setNodes([...Tree.setNode({ tree, destination, additions: node })]);
        rollbacks.add(() =>
          setNodes([...Tree.removeNode({ tree, keys: newIDString })]),
        );
        const [name, renamed] = await Text.asyncEdit(newIDString);
        if (!renamed || name === "") return false;
        return { ...value, key: newID.key, name };
      },
      [props],
    ),
    afterFailure: useCallback(
      async ({ status }: Flux.AfterFailureArgs<Group.CreateValue>) => {
        const { addStatus } = props;
        addStatus(status);
      },
      [props],
    ),
  });
  const {
    selection: { ids },
  } = props;
  return useCallback(
    () => update({ key: uuid.create(), name: "", parent: ids[0] }),
    [ids],
  );
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "group",
  icon: <Icon.Group />,
  canDrop: () => true,
  // This haul item allows the group to be dragged between nodes in the tree.
  haulItems: ({ id }) => [id],
  TreeContextMenu,
};
