// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Access } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Task } from "@/feature/ni/task";
import { type Device } from "@/platform/device";

export const RackMenuItems = ({
  rackKey,
}: Device.RackMenuItemsProps): ReactElement | null => {
  const canCreateTask = Access.useCreateGranted(task.TYPE_RESOURCE_ID);
  const toggleScanner = Task.useToggleScanner(rackKey);
  if (!canCreateTask) return null;
  return (
    <Menu.Item itemKey="toggleNIScanner" onClick={toggleScanner}>
      <Icon.Logo.NI />
      Toggle NI device scanner
    </Menu.Item>
  );
};
