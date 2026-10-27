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

import { Device } from "@/feature/ethercat/device";
import { Task } from "@/feature/ethercat/task";
import { Device as PlatformDevice } from "@/platform/device";

const TASK_CONTEXT_MENU_ITEMS = [
  PlatformDevice.createTaskContextMenuItem({
    itemKey: "ethercat.readTask",
    label: "Create read task",
    useCreate: Task.useCreateRead,
  }),
  PlatformDevice.createTaskContextMenuItem({
    itemKey: "ethercat.writeTask",
    label: "Create write task",
    useCreate: Task.useCreateWrite,
  }),
];

export const ContextMenuItems = (props: PlatformDevice.ContextMenuItemsProps) => {
  const keys = props.selection.ids.map((id) => id.key);
  const hasUpdatePermission = Access.useUpdateGranted(device.ontologyID(keys));
  const { update: toggleEnabled } = Device.useToggleEnabled();

  const { allDisabled, allEnabled } = Device.useEnabledState({ keys });

  const handleDisable = () => {
    toggleEnabled({ keys, enabled: false });
  };

  const handleEnable = () => {
    toggleEnabled({ keys, enabled: true });
  };

  return (
    <>
      <PlatformDevice.ConfigureMenuItem {...props} />
      <PlatformDevice.ChangeIdentifierMenuItem {...props} icon="Logo.EtherCAT" />
      <Menu.Divider />
      <PlatformDevice.TaskContextMenuItems {...props} items={TASK_CONTEXT_MENU_ITEMS} />
      <Menu.Divider />
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
