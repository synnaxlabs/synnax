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
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { type ReactElement } from "react";

/** Props for {@link GroupItems}. */
export interface GroupItemsProps {
  group: () => void;
  ungroup: () => void;
  canGroup: boolean;
  canUngroup: boolean;
}

/**
 * Renders the group and ungroup entries of a diagram context menu, wired to the
 * triggers from the diagram's useTriggers. An entry hides when it does not apply.
 * Render inside a {@link Menu}.
 */
export const GroupItems = ({
  group,
  ungroup,
  canGroup,
  canUngroup,
}: GroupItemsProps): ReactElement => (
  <>
    {canGroup && (
      <Menu.Item itemKey="group" onClick={group} triggerIndicator={Triggers.GROUP}>
        <Icon.Group />
        Group
      </Menu.Item>
    )}
    {canUngroup && (
      <Menu.Item
        itemKey="ungroup"
        onClick={ungroup}
        triggerIndicator={Triggers.UNGROUP}
      >
        <Icon.Ungroup />
        Ungroup
      </Menu.Item>
    )}
  </>
);
