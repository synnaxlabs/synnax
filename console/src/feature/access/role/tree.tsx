// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, ontology, user } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Tree as PTree } from "@synnaxlabs/lyra/tree";
import { Access, User } from "@synnaxlabs/pluto";

import { ContextMenu } from "@/platform/context-menu";
import { Tree } from "@/platform/tree";

const useDelete = Tree.createUseDelete({
  type: "Role",
  useUpdate: Access.Role.useDelete,
  convertKey: String,
});

const useRename = Tree.createUseRename({
  useUpdate: Access.Role.useRename,
  ontologyID: access.role.ontologyID,
  convertKey: String,
});

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    selection: { ids },
    state,
  } = props;
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const handleDelete = useDelete(props);
  const handleRename = useRename(props);
  const singleResource = ids.length === 1;
  const resources = Tree.useResources(state, ids);
  const hasInternal = resources.some((r) => r.data?.internal === true);
  return (
    <ContextMenu.Menu>
      {singleResource && hasUpdatePermission && !hasInternal && (
        <ContextMenu.RenameItem onClick={handleRename} />
      )}
      <Menu.Divider />
      {singleResource && <Tree.CopyPropertiesContextMenuItem {...props} />}
      <Menu.Divider />
      {hasDeletePermission && !hasInternal && (
        <ContextMenu.DeleteItem onClick={handleDelete} />
      )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const ROLE_ICONS: Record<string, Icon.ReactElement> = {
  Owner: <Icon.Settings />,
  Engineer: <Icon.Safety />,
  Operator: <Icon.Control />,
  Viewer: <Icon.Visible />,
  Driver: <Icon.Rack />,
};

const TreeItem = Tree.createItem({
  type: "role",
  icon: ({ name }) => ROLE_ICONS[name] ?? <Icon.Role />,
  ContextMenu: TreeContextMenu,
  hasChildren: true,
  canDrop: ({ items }) => {
    const resources = PTree.filterHaulItems(items);
    return (
      resources.length > 0 &&
      resources.every(
        ({ key }) => ontology.parseID(key).type === user.TYPE_ONTOLOGY_ID.type,
      ) &&
      User.filterHaulItems(items).every(({ data }) => !data.rootUser)
    );
  },
});

export const TREE_ITEMS = { role: TreeItem } satisfies Tree.Items;
