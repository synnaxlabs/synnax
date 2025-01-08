// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { user } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Menu as PMenu, Status, Tree } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type ReactElement } from "react";

import { Menu } from "@/components/menu";
import { Ontology } from "@/ontology";
import { Permissions } from "@/permissions";
import { useSelectHasPermission } from "@/user/selectors";

const useEditPermissions =
  (): ((props: Ontology.TreeContextMenuProps) => void) =>
  ({ placeLayout, selection }) =>
    placeLayout(
      Permissions.editLayout({ user: selection.resources[0].data as user.User }),
    );

const useDelete = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const confirm = Ontology.useConfirmDelete({ type: "User" });
  return useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    onMutate: async ({ state: { nodes, setNodes }, selection: { resources } }) => {
      if (!(await confirm(resources))) throw errors.CANCELED;
      const prevNodes = Tree.deepCopy(nodes);
      setNodes([
        ...Tree.removeNode({
          tree: nodes,
          keys: resources.map(({ id }) => id.toString()),
        }),
      ]);
      return prevNodes;
    },
    mutationFn: async ({ selection: { resources }, client }) =>
      await client.user.delete(resources.map(({ id }) => id.key)),
    onError: (e, { addStatus, state: { setNodes } }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      if (errors.CANCELED.matches(e)) return;
      Status.handleException(e, "Failed to delete users", addStatus);
    },
  }).mutate;
};

const TreeContextMenu: Ontology.TreeContextMenu = (props): ReactElement => {
  const {
    client,
    selection: { nodes, resources },
  } = props;
  const editPermissions = useEditPermissions();
  const handleDelete = useDelete();
  const handleSelect = {
    permissions: () => editPermissions(props),
    rename: () => Tree.startRenaming(nodes[0].key),
    delete: () => handleDelete(props),
  };
  const singleResource = resources.length === 1;
  const hasRootUser = resources.some((resource) => {
    const user = resource.data as user.User;
    return user.rootUser;
  });
  const isNotCurrentUser = resources[0].name !== client.props.username;
  const canEditPermissions = Permissions.useSelectCanEditPolicies();
  const canEditOrDelete = useSelectHasPermission();

  return (
    <PMenu.Menu onChange={handleSelect} level="small" iconSpacing="small">
      {singleResource && isNotCurrentUser && (
        <>
          {canEditPermissions && !hasRootUser && (
            <PMenu.Item itemKey="permissions" startIcon={<Icon.Access />}>
              Edit Permissions
            </PMenu.Item>
          )}
          {canEditOrDelete && (
            <>
              <PMenu.Item itemKey="rename" startIcon={<Icon.Rename />}>
                Change Username
              </PMenu.Item>
              <PMenu.Divider />
            </>
          )}
        </>
      )}
      {canEditOrDelete && !hasRootUser && (
        <>
          <Menu.DeleteItem />
          <PMenu.Divider />
        </>
      )}
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

const handleRename: Ontology.HandleTreeRename = {
  execute: async ({ client, id, name }) =>
    await client.user.changeUsername(id.key, name),
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: user.ONTOLOGY_TYPE,
  icon: <Icon.User />,
  hasChildren: true,
  allowRename: () => true,
  onRename: handleRename,
  haulItems: () => [],
  canDrop: () => false,
  onSelect: () => {},
  TreeContextMenu,
};
