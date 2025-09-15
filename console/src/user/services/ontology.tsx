// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, type user } from "@synnaxlabs/client";
import { Icon, Menu as PMenu, Text, User } from "@synnaxlabs/pluto";
import { useCallback, useMemo } from "react";

import { Menu } from "@/components";
import { Ontology } from "@/ontology";
import { Permissions } from "@/permissions";
import { useSelectHasPermission } from "@/user/selectors";

const editPermissions = ({
  placeLayout,
  selection: { ids },
  state: { getResource },
}: Ontology.TreeContextMenuProps) => {
  const user = getResource(ids[0]).data as user.User;
  const layout = Permissions.createEditLayout(user);
  placeLayout(layout);
};

const useDelete = ({
  selection: { ids },
  state: { getResource },
}: Ontology.TreeContextMenuProps): (() => void) => {
  const confirm = Ontology.useConfirmDelete({ type: "User" });
  const keys = useMemo(() => ids.map((id) => id.key), [ids]);
  const beforeUpdate = useCallback(
    async () => await confirm(getResource(ids)),
    [confirm, getResource, ids],
  );
  const { update } = User.useDelete({ beforeUpdate });
  return useCallback(() => update(keys), [update, keys]);
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    client,
    state: { getResource },
    selection: { ids },
  } = props;
  const handleDelete = useDelete(props);
  const handleSelect = {
    permissions: () => editPermissions(props),
    rename: () => Text.edit(ontology.idToString(ids[0])),
    delete: handleDelete,
  };
  const singleResource = ids.length === 1;
  const hasRootUser = ids.some((id) => {
    const user = getResource(id).data as user.User;
    return user.rootUser;
  });
  const isNotCurrentUser = getResource(ids[0]).name !== client.props.username;
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
    await client.users.changeUsername(id.key, name),
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "user",
  icon: <Icon.User />,
  allowRename: () => true,
  onRename: handleRename,
  TreeContextMenu,
};
