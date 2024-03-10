// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Drft, reloadWindow } from "@synnaxlabs/drift";
import { Icon } from "@synnaxlabs/media";
import { Menu } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

const HardReload = (props: Omit<Menu.ItemProps, "itemKey">): ReactElement => {
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

export const Item = {
  HardReload,
};
