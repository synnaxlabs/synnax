// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Menu as PMenu } from "@synnaxlabs/pluto";
import { Tree } from "@synnaxlabs/pluto/tree";
import { errors } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type ReactElement } from "react";
import { v4 as uuid } from "uuid";

import { Menu } from "@/components/menu";
import { useAsyncActionMenu } from "@/hooks/useAsyncAction";
import { Link } from "@/link";
import { Ontology } from "@/ontology";

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { nodes, parent, resources },
  } = props;
  const ungroup = useUngroupSelection();
  const createEmptyGroup = useCreateEmpty();
  const handleLink = Link.useCopyToClipboard();
  const onSelect = useAsyncActionMenu("group.menu", {
    ungroup: () => ungroup(props),
    rename: () => Tree.startRenaming(nodes[0].key),
    group: () => createEmptyGroup(props),
    link: () =>
      handleLink({
        name: resources[0].name,
        ontologyID: resources[0].id.payload,
      }),
  });
  const isDelete = nodes.every((n) => n.children == null || n.children.length === 0);
  const ungroupIcon = isDelete ? <Icon.Delete /> : <Icon.Group />;
  const singleResource = resources.length === 1;
  return (
    <PMenu.Menu onChange={onSelect} level="small" iconSpacing="small">
      {singleResource && (
        <>
          <Menu.RenameItem />
          <PMenu.Divider />
        </>
      )}
      <PMenu.Item itemKey="group" startIcon={<Icon.Group />}>
        New Group
      </PMenu.Item>
      {parent != null && (
        <PMenu.Item itemKey="ungroup" startIcon={ungroupIcon}>
          {/* TODO: Maybe we shouldn't force them into keeping the ontology tree like this? */}
          {isDelete ? "Delete" : "Ungroup"}
        </PMenu.Item>
      )}
      <PMenu.Divider />
      {singleResource && (
        <>
          <Link.CopyMenuItem />
          <PMenu.Divider />
        </>
      )}
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

export const UngroupMenuItem = (): ReactElement => (
  <PMenu.Item itemKey="ungroup" startIcon={<Icon.Group />}>
    Ungroup
  </PMenu.Item>
);

const createNewID = (): ontology.ID => new ontology.ID({ type: "group", key: uuid() });

export interface GroupMenuItemProps {
  selection: Ontology.TreeContextMenuProps["selection"];
}

export const GroupMenuItem = ({
  selection,
}: GroupMenuItemProps): ReactElement | null =>
  canGroupSelection(selection) ? (
    <PMenu.Item itemKey="group" startIcon={<Icon.Group />}>
      Group
    </PMenu.Item>
  ) : null;

const useUngroupSelection = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const mut = useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    mutationFn: async ({ client, selection, state: { nodes } }) => {
      if (selection.parent == null) return;
      for (const res of selection.resources) {
        const id = res.id;
        const children =
          Tree.findNode({ tree: nodes, key: id.toString() })?.children ?? [];
        const parentID = new ontology.ID(selection.parent.key as string);
        const childKeys = children.map((c) => new ontology.ID(c.key));
        await client.ontology.moveChildren(id, parentID, ...childKeys);
        await client.ontology.groups.delete(id.key);
      }
    },
    onError: async (
      { message },
      { selection, addStatus, state: { setNodes, nodes: prevNodes } },
    ) => {
      if (selection.parent == null || prevNodes == null) return;
      setNodes(prevNodes);
      addStatus({
        key: uuid(),
        variant: "error",
        message: "Failed to ungroup resources",
        description: message,
      });
    },
  });
  return (props: Ontology.TreeContextMenuProps) => {
    // Instead of using an onMutate argument to the useMutationHook, we do the eager
    // update beforehand so we can pass the previous nodes to the mutation. This lets
    // the mutationFn have access to the un-removed nodes while still allowing us
    // to eagerly update the UI.
    const {
      selection,
      state: { nodes, setNodes },
    } = props;
    if (selection.parent == null) return;
    // Sort the groups by depth that way deeper nested groups are ungrouped first.
    selection.resources.sort((a, b) => {
      const a_depth =
        selection.nodes.find((n) => n.key === a.id.toString())?.depth ?? 0;
      const b_depth =
        selection.nodes.find((n) => n.key === b.id.toString())?.depth ?? 0;
      return b_depth - a_depth;
    });
    const prevNodes = Tree.deepCopy(nodes);
    setNodes([
      ...selection.resources.reduce((acc, { id }) => {
        const children =
          Tree.findNode({ tree: nodes, key: id.toString() })?.children ?? [];
        acc = Tree.moveNode({
          tree: acc,
          destination: selection.parent?.key as string,
          keys: children.map((c) => c.key),
        });
        acc = Tree.removeNode({ tree: acc, keys: id.toString() });
        return acc;
      }, nodes),
    ]);
    mut.mutate({ ...props, state: { ...props.state, nodes: prevNodes } });
  };
};

export const canGroupSelection = (
  selection: Ontology.TreeContextMenuProps["selection"],
): boolean => getAllNodesOfMinDepth(selection.nodes).length > 1;

const getAllNodesOfMinDepth = (
  nodes: Tree.NodeWithPosition[],
): Tree.NodeWithPosition[] => {
  if (nodes.length === 0) return [];
  const depths = nodes.map(({ depth }) => depth).sort((a, b) => a - b);
  const minDepth = depths[0];
  return nodes.filter(({ depth }) => depth === minDepth);
};

export const useCreateEmpty = (): ((
  props: Ontology.TreeContextMenuProps,
) => Promise<void>) => {
  const mut = useMutation<
    void,
    Error,
    Ontology.TreeContextMenuProps & { newID: ontology.ID }
  >({
    onMutate: async ({
      services,
      selection: { resources },
      state: { nodes, setNodes, expand },
      newID,
    }) => {
      if (resources.length === 0) return;
      const resource = resources[resources.length - 1];
      const res: ontology.Resource = {
        key: newID.toString(),
        id: newID,
        name: "",
      };
      expand(resource.id.toString());
      const newGroupNode = Ontology.toTreeNode(services, res);
      setNodes([
        ...Tree.setNode({
          tree: nodes,
          destination: resource.id.toString(),
          additions: newGroupNode,
        }),
      ]);
    },
    mutationFn: async ({ client, selection: { resources }, newID }) => {
      const resource = resources[resources.length - 1];
      const [name, renamed] = await Tree.asyncRename(newID.toString());
      if (!renamed) throw errors.CANCELED;
      await client.ontology.groups.create(resource.id, name, newID.key);
    },
    onError: async (
      { message },
      { state: { nodes, setNodes }, addStatus, selection, newID },
    ) => {
      if (selection.resources.length === 0) return;
      if (!errors.CANCELED.matches(message))
        addStatus({
          key: uuid(),
          variant: "error",
          message: "Failed to create group",
          description: message,
        });
      setNodes([...Tree.removeNode({ tree: nodes, keys: newID.toString() })]);
    },
  });
  return async (props: Ontology.TreeContextMenuProps) =>
    mut.mutate({ ...props, newID: createNewID() });
};

const getResourcesToGroup = (
  selection: Ontology.TreeContextMenuProps["selection"],
): ontology.ID[] => {
  const nodesOfMinDepth = getAllNodesOfMinDepth(selection.nodes);
  const nodesOfMinDepthKeys = nodesOfMinDepth.map(({ key }) => key);
  return selection.resources
    .filter(({ id }) => nodesOfMinDepthKeys.includes(id.toString()))
    .map(({ id }) => id);
};

export const useCreateFromSelection = (): ((
  props: Ontology.TreeContextMenuProps,
) => void) => {
  const mut = useMutation<
    void,
    Error,
    Ontology.TreeContextMenuProps & { newID: ontology.ID },
    Tree.Node[]
  >({
    onMutate: async ({
      selection,
      state: { nodes, setNodes, setSelection },
      newID,
    }) => {
      if (selection.parent == null) return;
      const resourcesToGroup = getResourcesToGroup(selection);
      const prevNodes = Tree.deepCopy(nodes);
      let nextNodes = Tree.setNode({
        tree: nodes,
        destination: selection.parent.key,
        additions: {
          key: newID.toString(),
          icon: <Icon.Group />,
          children: [],
          name: "",
          allowRename: true,
        },
      });
      nextNodes = Tree.moveNode({
        tree: nodes,
        destination: newID.toString(),
        keys: resourcesToGroup.map((id) => id.toString()),
      });
      setNodes([...nextNodes]);
      setSelection([newID.toString()]);
      return prevNodes;
    },
    mutationFn: async ({ client, selection, newID }) => {
      if (selection.parent == null) return;
      const [groupName, renamed] = await Tree.asyncRename(newID.toString());
      if (!renamed) throw errors.CANCELED;
      const resourcesToGroup = getResourcesToGroup(selection);
      const parentID = new ontology.ID(selection.parent.key);
      await client.ontology.groups.create(parentID, groupName, newID.key);
      await client.ontology.moveChildren(parentID, newID, ...resourcesToGroup);
    },
    onError: async ({ message }, { state: { setNodes }, addStatus }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      if (errors.CANCELED.matches(message)) return;
      addStatus({
        key: uuid(),
        variant: "error",
        message: "Failed to group resources",
        description: message,
      });
    },
  });
  return (props: Ontology.TreeContextMenuProps) =>
    mut.mutate({ ...props, newID: createNewID() });
};

const handleRename: Ontology.HandleTreeRename = {
  execute: async ({ client, id, name }) => {
    try {
      await client.ontology.groups.rename(id.key, name);
    } catch (e) {
      // We check for this because the rename might be a side effect of creating
      // a new group, in which case the group might not exist yet. This is fine
      // and we don't want to throw an error.
      if (!NotFoundError.matches(e)) throw e;
    }
  },
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: "group",
  icon: <Icon.Group />,
  hasChildren: true,
  onRename: handleRename,
  canDrop: () => true,
  onSelect: () => {},
  haulItems: () => [],
  allowRename: () => true,
  onMosaicDrop: () => {},
  TreeContextMenu,
};
