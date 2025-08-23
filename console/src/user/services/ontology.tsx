// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, type user } from "@synnaxlabs/client";
import { Icon, Menu as PMenu, Text, Tree } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";

import { Menu } from "@/components";
import { Ontology } from "@/ontology";
import { Permissions } from "@/permissions";
import { useSelectHasPermission } from "@/user/selectors";

const editPermissions = ({
  placeLayout,
  selection: { resourceIDs },
  state: { getResource },
}: Ontology.TreeContextMenuProps) => {
  const user = getResource(resourceIDs[0]).data as user.User;
  const layout = Permissions.createEditLayout(user);
  placeLayout(layout);
};

const useDelete = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const confirm = Ontology.useConfirmDelete({ type: "User" });
  return useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    onMutate: async ({
      state: { nodes, setNodes, getResource },
      selection: { resourceIDs },
    }) => {
      const resources = getResource(resourceIDs);
      if (!(await confirm(resources))) throw new errors.Canceled();
      const prevNodes = Tree.deepCopy(nodes);
      setNodes([
        ...Tree.removeNode({
          tree: nodes,
          keys: resourceIDs.map((id) => ontology.idToString(id)),
        }),
      ]);
      return prevNodes;
    },
    mutationFn: async ({ selection: { resourceIDs }, client }) =>
      await client.user.delete(resourceIDs.map((id) => id.key)),
    onError: (e, { handleError, state: { setNodes } }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      if (errors.Canceled.matches(e)) return;
      handleError(e, "Failed to delete users");
    },
  }).mutate;
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    client,
    state: { getResource },
    selection: { resourceIDs },
  } = props;
  const handleDelete = useDelete();
  const handleSelect = {
    permissions: () => editPermissions(props),
    rename: () => Text.edit(ontology.idToString(resourceIDs[0])),
    delete: () => handleDelete(props),
  };
  const singleResource = resourceIDs.length === 1;
  const hasRootUser = resourceIDs.some((id) => {
    const user = getResource(id).data as user.User;
    return user.rootUser;
  });
  const isNotCurrentUser = getResource(resourceIDs[0]).name !== client.props.username;
  const canEditPermissions = Permissions.useSelectCanEditPolicies();
  const canEditOrDelete = useSelectHasPermission();

  return (
    <PMenu.Menu onChange={handleSelect} level="small" gap="small">
      {singleResource && isNotCurrentUser && (
        <>
          {canEditPermissions && !hasRootUser && (
            <PMenu.Item itemKey="permissions">
              <Icon.Access />
              Edit Permissions
            </PMenu.Item>
          )}
          {canEditOrDelete && (
            <>
              <PMenu.Item itemKey="rename">
                <Icon.Rename />
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
  ...Ontology.NOOP_SERVICE,
  type: "user",
  icon: <Icon.User />,
  allowRename: () => true,
  onRename: handleRename,
  TreeContextMenu,
};
