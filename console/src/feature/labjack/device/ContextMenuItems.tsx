// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Menu } from "@synnaxlabs/lyra/menu";

import { ADAPTER } from "@/feature/labjack/device/hardware";
import { Task } from "@/feature/labjack/task";
import { Device as PlatformDevice } from "@/platform/device";

const TASK_CONTEXT_MENU_ITEMS = [
  PlatformDevice.createTaskContextMenuItem({
    itemKey: "labjack.readTask",
    label: "Create read task",
    useCreate: Task.useCreateRead,
  }),
  PlatformDevice.createTaskContextMenuItem({
    itemKey: "labjack.writeTask",
    label: "Create write task",
    useCreate: Task.useCreateWrite,
  }),
];

const ConfigureMenuItem = PlatformDevice.createConfigureMenuItem(ADAPTER);

const ChangeIdentifierMenuItem = PlatformDevice.createChangeIdentifierMenuItem({
  adapter: ADAPTER,
  icon: "Logo.LabJack",
});

export const ContextMenuItems = (props: PlatformDevice.ContextMenuItemsProps) => (
  <>
    <ConfigureMenuItem {...props} />
    <ChangeIdentifierMenuItem {...props} />
    <Menu.Divider />
    <PlatformDevice.TaskContextMenuItems {...props} items={TASK_CONTEXT_MENU_ITEMS} />
  </>
);
