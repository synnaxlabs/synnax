// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Menu } from "@synnaxlabs/lyra/menu";

import { Task } from "@/feature/sift/task";
import { Device } from "@/platform/device";

const TASK_CONTEXT_MENU_ITEMS = [
  Device.createTaskContextMenuItem({
    itemKey: "sift.streamTask",
    label: "Create stream task",
    useCreate: Task.useCreateStream,
  }),
];

export const ContextMenuItems = (props: Device.ContextMenuItemsProps) => (
  <>
    <Device.EditConnectionMenuItem {...props} />
    <Menu.Divider />
    <Device.TaskContextMenuItems {...props} items={TASK_CONTEXT_MENU_ITEMS} />
  </>
);
