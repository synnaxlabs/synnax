// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Icon, Menu } from "@synnaxlabs/pluto";

import { type Tree } from "@/platform/tree";

export interface EditConnectionMenuItemProps extends Pick<
  Tree.ContextMenuProps,
  "selection"
> {
  onConfigure: (deviceKey: device.Key) => void;
}

export const EditConnectionMenuItem = ({
  onConfigure,
  selection: { ids },
}: EditConnectionMenuItemProps) => {
  if (ids.length !== 1) return null;
  const handleClick = () => onConfigure(ids[0].key);
  return (
    <Menu.Item itemKey="editConnection" onClick={handleClick}>
      <Icon.Edit />
      Edit connection
    </Menu.Item>
  );
};
