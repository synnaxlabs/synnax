// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Menu as PMenu, Tree } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Menu } from "@/components/menu";
import { Ontology } from "@/ontology";
import { Permissions } from "@/permissions";

import { overviewLayout } from "../Overview";

const useSetPermissions =
  (): ((props: Ontology.TreeContextMenuProps) => void) =>
  ({ placeLayout, selection: { resources } }) =>
    placeLayout(
      Permissions.layout({
        user: {
          username: resources[0].name,
          key: resources[0].id.key,
        },
      }),
    );

const TreeContextMenu: Ontology.TreeContextMenu = (props): ReactElement => {
  const {
    selection: { nodes, resources },
  } = props;
  const setPermissions = useSetPermissions();
  const handleSelect = {
    permissions: () => setPermissions(props),
    rename: () => Tree.startRenaming(nodes[0].key),
  };
  const singleResource = resources.length === 1;

  return (
    <PMenu.Menu onChange={handleSelect} level="small" iconSpacing="small">
      {singleResource && (
        <>
          <PMenu.Item itemKey="permissions" startIcon={<Icon.Access />}>
            Set Permissions
          </PMenu.Item>
          <PMenu.Item itemKey="rename" startIcon={<Icon.Rename />}>
            Change Username
          </PMenu.Item>
          {/*TODO: Change Name*/}
          <PMenu.Divider />
        </>
      )}
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

const handleSelect: Ontology.HandleSelect = async ({
  selection,
  client,
  placeLayout,
}): Promise<void> => {
  if (selection.length === 0) return;
  const user = await client.user.retrieve(selection[0].id.key);
  placeLayout({ ...overviewLayout, name: user.username, key: user.key });
};

const handleRename: Ontology.HandleTreeRename = {
  execute: async ({ client, id, name }) =>
    await client.user.changeUsername(id.key, name),
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: "user",
  icon: <Icon.User />,
  hasChildren: true,
  allowRename: () => true,
  onRename: handleRename,
  haulItems: () => [],
  canDrop: () => false,
  onSelect: handleSelect,
  TreeContextMenu,
};
