// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Access, Device } from "@synnaxlabs/pluto";

import { type Tree } from "@/platform/tree";

export interface ConfigureMenuItemProps extends Pick<
  Tree.ContextMenuProps,
  "selection"
> {
  onConfigure: (deviceKey: device.Key) => void;
}

export const ConfigureMenuItem = ({
  onConfigure,
  selection: { ids },
}: ConfigureMenuItemProps) => {
  const { data: dev } = Device.useResult({ key: ids[0].key });
  const hasUpdatePermission = Access.useUpdateGranted(device.resourceID(ids[0].key));
  if (ids.length !== 1 || dev == null || dev.configured || !hasUpdatePermission)
    return null;
  const handleClick = () => onConfigure(dev.key);
  return (
    <Menu.Item itemKey="configure" onClick={handleClick}>
      <Icon.Hardware />
      Configure
    </Menu.Item>
  );
};
