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
import { type ReactElement } from "react";

import { useOpen } from "@/app/settings/tab";
import { Panel } from "@/platform/panel";

/** The gear in the left bar that opens Settings. */
export const Button = (): ReactElement | null => {
  const open = useOpen();
  const canOpen = Panel.useCanOpenTab();
  if (!canOpen) return null;
  return (
    <Menu.Menu onChange={open}>
      <Menu.Item itemKey="settings" size="medium" aria-label="Settings">
        <Icon.Settings />
      </Menu.Item>
    </Menu.Menu>
  );
};
