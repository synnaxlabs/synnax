// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/labjack/device";
import { Task } from "@/hardware/labjack/task";
import { type Ontology } from "@/ontology";

const TASK_CONTEXT_MENU_ITEM_CONFIGS: Common.DeviceServices.TaskContextMenuItemConfig[] =
  [
    {
      itemKey: "labjack.readTask",
      label: "Create read task",
      layout: Task.READ_LAYOUT,
    },
    {
      itemKey: "labjack.writeTask",
      label: "Create write task",
      layout: Task.WRITE_LAYOUT,
    },
  ];

export const ContextMenuItems = (props: Ontology.TreeContextMenuProps) => (
  <Common.DeviceServices.ContextMenuItems
    {...props}
    configureLayout={Device.CONFIGURE_LAYOUT}
    taskContextMenuItemConfigs={TASK_CONTEXT_MENU_ITEM_CONFIGS}
  />
);
