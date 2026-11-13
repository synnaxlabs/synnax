// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Menu } from "@synnaxlabs/lyra/menu";

import { ADAPTER } from "@/feature/ni/device/hardware";
import { Task } from "@/feature/ni/task";
import { Device as PlatformDevice } from "@/platform/device";

const TASK_CONTEXT_MENU_ITEMS = [
  PlatformDevice.createTaskContextMenuItem({
    itemKey: "ni.analogReadTask",
    label: "Create analog read task",
    useCreate: Task.useCreateAnalogRead,
  }),
  PlatformDevice.createTaskContextMenuItem({
    itemKey: "ni.analogWriteTask",
    label: "Create analog write task",
    useCreate: Task.useCreateAnalogWrite,
  }),
  PlatformDevice.createTaskContextMenuItem({
    itemKey: "ni.counterReadTask",
    label: "Create counter read task",
    useCreate: Task.useCreateCounterRead,
  }),
  PlatformDevice.createTaskContextMenuItem({
    itemKey: "ni.digitalReadTask",
    label: "Create digital read task",
    useCreate: Task.useCreateDigitalRead,
  }),
  PlatformDevice.createTaskContextMenuItem({
    itemKey: "ni.digitalWriteTask",
    label: "Create digital write task",
    useCreate: Task.useCreateDigitalWrite,
  }),
];

const ConfigureMenuItem = PlatformDevice.createConfigureMenuItem(ADAPTER);

const ChangeIdentifierMenuItem = PlatformDevice.createChangeIdentifierMenuItem({
  adapter: ADAPTER,
  icon: "Logo.NI",
});

export const ContextMenuItems = (props: PlatformDevice.ContextMenuItemsProps) => (
  <>
    <ConfigureMenuItem {...props} />
    <ChangeIdentifierMenuItem {...props} />
    <Menu.Divider />
    <PlatformDevice.TaskContextMenuItems {...props} items={TASK_CONTEXT_MENU_ITEMS} />
  </>
);
