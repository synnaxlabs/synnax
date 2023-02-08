// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Menu } from "@synnaxlabs/pluto";
import type { MenuItemProps } from "@synnaxlabs/pluto";
import { Icon } from "../Icon";

const HardReload = (props: MenuItemProps) => (
    <Menu.Item 
      onClick={() => window.location.reload()}
      startIcon={<Icon.Refresh />}
      size="small"
      {...props}
    >
      Hard Reload
    </Menu.Item>
);

export const Item = {
  HardReload,
}

