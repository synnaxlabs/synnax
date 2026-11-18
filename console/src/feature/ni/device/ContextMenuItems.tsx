// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Menu } from "@synnaxlabs/lyra/menu";

import { suggestIdentifiers } from "@/feature/ni/device/identifier";
import { ZERO_PROPERTIES } from "@/feature/ni/device/types";
import { Task } from "@/feature/ni/task";
import { Device as PlatformDevice } from "@/platform/device";

const CONFIGURE_MANY = { zeroProperties: ZERO_PROPERTIES, suggestIdentifiers };

const TASK_CONTEXT_MENU_ITEMS = [
  PlatformDevice.createTaskContextMenuItem({
    itemKey: "ni.analogReadTask",
    label: "Create analog read task",
    useCreate: Task.useCreateAnalogRead,
    configureMany: CONFIGURE_MANY,
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
    configureMany: CONFIGURE_MANY,
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

export const ContextMenuItems = (props: PlatformDevice.ContextMenuItemsProps) => (
  <>
    <PlatformDevice.ConfigureMenuItem {...props} />
    <PlatformDevice.ChangeIdentifierMenuItem {...props} icon="Logo.NI" />
    <Menu.Divider />
    <PlatformDevice.TaskContextMenuItems {...props} items={TASK_CONTEXT_MENU_ITEMS} />
  </>
);
