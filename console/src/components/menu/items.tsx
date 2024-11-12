// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { reloadWindow } from "@synnaxlabs/drift";
import { Icon } from "@synnaxlabs/media";
import { Menu } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";
import { useDispatch } from "react-redux";

export const HardReloadItem = (
  props: Omit<Menu.ItemProps, "itemKey">,
): ReactElement => {
  const d = useDispatch();
  return (
    <Menu.Item
      onClick={() => d(reloadWindow({}))}
      startIcon={<Icon.Refresh />}
      size="small"
      itemKey="hardReload"
      {...props}
    >
      Hard Reload
    </Menu.Item>
  );
};

export const RenameItem = (): ReactElement => (
  <Menu.Item itemKey="rename" size="small" startIcon={<Icon.Rename />}>
    Rename
  </Menu.Item>
);

export const DeleteItem = (): ReactElement => (
  <Menu.Item itemKey="delete" startIcon={<Icon.Delete />} size="small">
    Delete
  </Menu.Item>
);
