// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";

import { useTare, useTareable } from "@/platform/task/useTare";

export interface TareItemProps {
  /** The Synnax channels of the selected items, with 0 for an item with none. */
  channels: channel.Key[];
}

/**
 * Tares the float channels among the given ones while the task runs. Renders nothing
 * for a snapshot, a stopped task, or no float channel.
 */
export const TareItem = ({ channels }: TareItemProps) => {
  const tareable = useTareable(channels);
  const tare = useTare();
  if (tareable == null || !tareable.running || tareable.floats.length === 0)
    return null;
  const { floats } = tareable;
  return (
    <Menu.Item itemKey="tare" onClick={() => tare(floats)}>
      <Icon.Tare />
      Tare
    </Menu.Item>
  );
};
