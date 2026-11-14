// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Access } from "@synnaxlabs/pluto";
import { type ComponentType, type ReactElement } from "react";

import { type Adapter } from "@/platform/device/adapter";
import { type Tree } from "@/platform/tree";

export interface ConfigureMenuItemProps extends Pick<
  Tree.ContextMenuProps,
  "selection"
> {
  onConfigure: (deviceKey: string) => void;
}

/** Creates the menu item that opens the configure form of an unconfigured device. */
export const createConfigureMenuItem = ({
  use: useConfigurable,
  resourceID,
}: Adapter): ComponentType<ConfigureMenuItemProps> => {
  const ConfigureMenuItem = ({
    onConfigure,
    selection: { ids },
  }: ConfigureMenuItemProps): ReactElement | null => {
    const dev = useConfigurable(ids[0].key);
    const hasUpdatePermission = Access.useUpdateGranted(resourceID(ids[0].key));
    if (ids.length !== 1 || dev.configured || !hasUpdatePermission) return null;
    const handleClick = () => onConfigure(dev.key);
    return (
      <Menu.Item itemKey="configure" onClick={handleClick}>
        <Icon.Hardware />
        Configure
      </Menu.Item>
    );
  };
  return ConfigureMenuItem;
};
