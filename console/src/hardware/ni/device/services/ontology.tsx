// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/ni/device";
import { Task } from "@/hardware/ni/task";
import { type Ontology } from "@/ontology";

const TASK_CONTEXT_MENU_ITEM_CONFIGS: Common.DeviceServices.TaskContextMenuItemConfig[] =
  [
    {
      itemKey: "ni.analogReadTask",
      label: "Create Analog Read Task",
      layout: Task.ANALOG_READ_LAYOUT,
    },
    {
      itemKey: "ni.analogWriteTask",
      label: "Create Analog Write Task",
      layout: Task.ANALOG_WRITE_LAYOUT,
    },
    {
      itemKey: "ni.digitalReadTask",
      label: "Create Digital Read Task",
      layout: Task.DIGITAL_READ_LAYOUT,
    },
    {
      itemKey: "ni.digitalWriteTask",
      label: "Create Digital Write Task",
      layout: Task.DIGITAL_WRITE_LAYOUT,
    },
  ];

export const ContextMenuItems = (props: Ontology.TreeContextMenuProps) => (
  <Common.DeviceServices.ContextMenuItems
    {...props}
    deviceConfigLayout={Device.CONFIGURE_LAYOUT}
    taskContextMenuItemConfigs={TASK_CONTEXT_MENU_ITEM_CONFIGS}
  />
);
