// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Menu } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { type Session } from "@/session";

export interface SnapshotMenuItemProps {
  range?: Session.Range.State | null;
  onClick?: () => void;
}

export const SnapshotMenuItem = ({
  range,
  onClick,
}: SnapshotMenuItemProps): ReactElement | null =>
  range?.persisted === true ? (
    <Menu.Item itemKey="rangeSnapshot" onClick={onClick}>
      <Icon.Snapshot />
      Snapshot to {range.name}
    </Menu.Item>
  ) : null;
