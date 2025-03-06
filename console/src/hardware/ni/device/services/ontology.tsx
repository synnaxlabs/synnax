// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Menu, Status, Synnax, TimeSpan } from "@synnaxlabs/pluto";
import { useMutation } from "@tanstack/react-query";

import { NULL_CLIENT_ERROR } from "@/errors";
import { Common } from "@/hardware/common";
import { Device } from "@/hardware/ni/device";
import { Task } from "@/hardware/ni/task";
import { SCAN_TASK_NAME } from "@/hardware/ni/task/types";
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

const ResetDeviceMenuItem = ({ deviceKey }: { deviceKey: string }) => {
  const client = Synnax.use();
  const handleException = Status.useExceptionHandler();
  const { mutate } = useMutation({
    mutationFn: async (key: string) => {
      if (client == null) throw NULL_CLIENT_ERROR;
      const device = await client.hardware.devices.retrieve(key);
      const scanTask = await client.hardware.tasks.retrieve(device.rack, {
        types: ["ni_scanner"],
      });
      const t = scanTask[0];
      await t.executeCommandSync(
        "reset_device",
        { deviceKeys: [key] },
        TimeSpan.seconds(10),
      );
    },
    onError: handleException,
  });
  return (
    <Menu.Item
      itemKey="ni.resetDevice"
      startIcon={<Icon.Refresh />}
      onClick={() => mutate(deviceKey)}
    />
  );
};

export const ContextMenuItems = (props: Ontology.TreeContextMenuProps) => (
  <>
    <Common.DeviceServices.ContextMenuItems
      {...props}
      configureLayout={Device.CONFIGURE_LAYOUT}
      taskContextMenuItemConfigs={TASK_CONTEXT_MENU_ITEM_CONFIGS}
    />
    <ResetDeviceMenuItem deviceKey={props.selection.resources[0].id.key} />
  </>
);
