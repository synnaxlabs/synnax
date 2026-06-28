// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Menu } from "@synnaxlabs/pluto";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/modbus/device";
import { Task } from "@/hardware/modbus/task";
import { type Ontology } from "@/ontology";

const TASK_CONTEXT_MENU_ITEM_CONFIGS: Common.DeviceServices.TaskContextMenuItemConfig[] =
  [
    { itemKey: "modbus.readTask", label: "Create read task", layout: Task.READ_LAYOUT },
    {
      itemKey: "modbus.writeTask",
      label: "Create write task",
      layout: Task.WRITE_LAYOUT,
    },
  ];

export const ContextMenuItems = (props: Ontology.TreeContextMenuProps) => {
  const connect = Device.useConnectModal();
  const onConfigure = (deviceKey: string) => connect({ deviceKey });
  return (
    <>
      <Common.DeviceServices.EditConnectionMenuItem
        {...props}
        onConfigure={onConfigure}
      />
      <Menu.Divider />
      <Common.DeviceServices.TaskContextMenuItems
        {...props}
        onConfigure={onConfigure}
        taskContextMenuItemConfigs={TASK_CONTEXT_MENU_ITEM_CONFIGS}
      />
    </>
  );
};
