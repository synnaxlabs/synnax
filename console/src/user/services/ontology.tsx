// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Menu as PMenu } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Menu } from "@/components/menu";
import { Ontology } from "@/ontology";
import { Permissions } from "@/permissions";

const useSetPermissions =
  (): ((props: Ontology.TreeContextMenuProps) => void) =>
  ({ placeLayout, selection: { resources } }) => {
    placeLayout(
      Permissions.layout({
        user: {
          username: resources[0].name,
          key: resources[0].id.key,
        },
      }),
    );
  };

const TreeContextMenu: Ontology.TreeContextMenu = (props): ReactElement => {
  const {
    selection: { resources },
  } = props;
  const setPermissions = useSetPermissions();
  const handleSelect = { permissions: () => setPermissions(props) };
  const singleResource = resources.length === 1;
  const canEditPermissions = Permissions.useSelectAdmin();
  const showSetPermissions = singleResource && canEditPermissions;
  return (
    <PMenu.Menu onChange={handleSelect} level="small" iconSpacing="small">
      {showSetPermissions && (
        <>
          <PMenu.Item itemKey="permissions" startIcon={<Icon.Access />}>
            Set Permissions
          </PMenu.Item>
          <PMenu.Divider />
        </>
      )}
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: "user",
  icon: <Icon.User />,
  hasChildren: true,
  allowRename: () => false,
  haulItems: () => [],
  canDrop: () => false,
  onSelect: () => {},
  TreeContextMenu,
};
