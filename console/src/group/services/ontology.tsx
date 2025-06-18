// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, NotFoundError, ontology } from "@synnaxlabs/client";
import { Icon, Menu as PMenu, Tree } from "@synnaxlabs/pluto";
import { errors, uuid } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";

import { Cluster } from "@/cluster";
import { Menu } from "@/components/menu";
import { MenuItem } from "@/group/MenuItem";
import { useCreateFromSelection } from "@/group/useCreateFromSelection";
import { useAsyncActionMenu } from "@/hooks/useAsyncAction";
import { Link } from "@/link";
import { Ontology } from "@/ontology";

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { nodes, resources },
  } = props;
  const ungroup = useUngroupSelection();
  const createEmptyGroup = useCreateEmpty();
  const createFromSelection = useCreateFromSelection();
  const handleLink = Cluster.useCopyLinkToClipboard();
  const onSelect = useAsyncActionMenu({
    ungroup: () => ungroup(props),
    rename: () => Tree.startRenaming(nodes[0].key),
    newGroup: () => createEmptyGroup(props),
    group: () => createFromSelection(props),
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
          <PMenu.Item itemKey="newGroup" startIcon={<Icon.Group />}>
            New Group
          </PMenu.Item>
        </>
      )}
      <MenuItem selection={props.selection} />
      <PMenu.Item itemKey="ungroup" startIcon={ungroupIcon}>
        {/* TODO: Maybe we shouldn't force them into keeping the ontology tree like this? */}
        {isDelete ? "Delete" : "Ungroup"}
      </PMenu.Item>
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

const useUngroupSelection = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const mut = useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    mutationFn: async ({ client, selection, state: { nodes } }) => {
      if (selection.parentID == null) return;
      for (const res of selection.resources) {
        const id = res.id;
        const children =
          Tree.findNode({ tree: nodes, key: id.toString() })?.children ?? [];
        const parentID = selection.parentID;
        const childKeys = children.map((c) => new ontology.ID(c.key));
        await client.ontology.moveChildren(id, parentID, ...childKeys);
        await client.ontology.groups.delete(id.key);
      }
    },
    onError: async (
      e,
      { selection, handleError, state: { setNodes, nodes: prevNodes } },
    ) => {
      if (selection.parentID == null || prevNodes == null) return;
      setNodes(prevNodes);
      handleError(e, "Failed to ungroup resources");
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
    if (selection.parentID == null) return;
    // Sort the groups by depth that way deeper nested groups are ungrouped first.
    selection.nodes.sort((a, b) => b.depth - a.depth);
    const prevNodes = Tree.deepCopy(nodes);
    setNodes([
      ...selection.nodes.reduce((acc, { key }) => {
        const children = Tree.findNode({ tree: nodes, key })?.children ?? [];
        acc = Tree.moveNode({
          tree: acc,
          destination:
            selection.parentID.toString() === selection.rootID.toString()
              ? null
              : selection.parentID.toString(),
          keys: children.map((c) => c.key),
        });
        acc = Tree.removeNode({ tree: acc, keys: key });
        return acc;
      }, nodes),
    ]);
    mut.mutate({ ...props, state: { ...props.state, nodes: prevNodes } });
  };
};

const useCreateEmpty = (): ((
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
      const res: ontology.Resource = { key: newID.toString(), id: newID, name: "" };
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
      if (!renamed) throw new errors.Canceled();
      await client.ontology.groups.create(resource.id, name, newID.key);
    },
    onError: async (
      e,
      { state: { nodes, setNodes }, handleError, selection, newID },
    ) => {
      if (selection.resources.length === 0) return;
      if (!errors.Canceled.matches(e)) handleError(e, "Failed to create group");
      setNodes([...Tree.removeNode({ tree: nodes, keys: newID.toString() })]);
    },
  });
  return async (props: Ontology.TreeContextMenuProps) =>
    mut.mutate({ ...props, newID: group.ontologyID(uuid.create()) });
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
  ...Ontology.NOOP_SERVICE,
  type: group.ONTOLOGY_TYPE,
  icon: <Icon.Group />,
  canDrop: () => true,
  // This haul item allows the group to be dragged between nodes in the tree.
  haulItems: ({ id }) => [id],
  allowRename: () => true,
  onRename: handleRename,
  TreeContextMenu,
};
