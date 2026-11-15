// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Access } from "@synnaxlabs/pluto";

import {
  useEnabledState,
  useResultSlave,
  useToggleEnabled,
} from "@/feature/ethercat/device/queries";
import { Task } from "@/feature/ethercat/task";
import { Device } from "@/platform/device";

interface ConfigureItemProps extends Pick<Device.ContextMenuItemsProps, "onConfigure"> {
  deviceKey: device.Key;
}

const ConfigureItem = ({ deviceKey, onConfigure }: ConfigureItemProps) => (
  <Menu.Item itemKey="ethercat.configure" onClick={() => onConfigure(deviceKey)}>
    <Icon.Hardware />
    Configure
  </Menu.Item>
);

const READ_TASK_ITEM = Device.createTaskContextMenuItem({
  itemKey: "ethercat.readTask",
  label: "Create read task",
  useCreate: Task.useCreateRead,
});

const WRITE_TASK_ITEM = Device.createTaskContextMenuItem({
  itemKey: "ethercat.writeTask",
  label: "Create write task",
  useCreate: Task.useCreateWrite,
});

export const ContextMenuItems = (props: Device.ContextMenuItemsProps) => {
  const keys = props.selection.ids.map((id) => id.key);
  const hasUpdatePermission = Access.useUpdateGranted(device.resourceID(keys));
  const { update: toggleEnabled } = useToggleEnabled();

  const { allDisabled, allEnabled } = useEnabledState({ keys });
  const { data: first } = useResultSlave({ key: keys[0] });
  const pdos = first?.properties.pdos;
  const taskItems = [
    ...(pdos != null && pdos.inputs.length > 0 ? [READ_TASK_ITEM] : []),
    ...(pdos != null && pdos.outputs.length > 0 ? [WRITE_TASK_ITEM] : []),
  ];

  const handleDisable = () => {
    toggleEnabled({ keys, enabled: false });
  };

  const handleEnable = () => {
    toggleEnabled({ keys, enabled: true });
  };

  return (
    <>
      {hasUpdatePermission && keys.length === 1 && (
        <ConfigureItem deviceKey={keys[0]} onConfigure={props.onConfigure} />
      )}
      <Menu.Divider />
      {taskItems.length > 0 && (
        <>
          <Device.TaskContextMenuItems {...props} items={taskItems} />
          <Menu.Divider />
        </>
      )}
      {hasUpdatePermission && !allDisabled && (
        <Menu.Item itemKey="ethercat.disable" onClick={handleDisable}>
          <Icon.Disable />
          Disable
        </Menu.Item>
      )}
      {hasUpdatePermission && !allEnabled && (
        <Menu.Item itemKey="ethercat.enable" onClick={handleEnable}>
          <Icon.Enable />
          Enable
        </Menu.Item>
      )}
    </>
  );
};
