// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Menu } from "@synnaxlabs/pluto";

import { type Range } from "@/range/slice";

export interface SnapshotMenuItemProps {
  range?: Range | null;
}

export const SnapshotMenuItem = ({ range }: SnapshotMenuItemProps) =>
  range?.persisted === true && (
    <Menu.Item itemKey="rangeSnapshot" startIcon={<Icon.Snapshot />}>
      Snapshot to {range.name}
    </Menu.Item>
  );
