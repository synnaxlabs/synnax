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

import { useEnabledState, useToggleEnabled } from "@/feature/ethercat/device/queries";
import { useSyncModeModal } from "@/feature/ethercat/device/useSyncModeModal";
import { Task } from "@/feature/ethercat/task";
import { FLAGS } from "@/flags";
import { Device } from "@/platform/device";

interface SyncModeItemProps {
  deviceKey: device.Key;
}

const SyncModeItem = ({ deviceKey }: SyncModeItemProps) => {
  const openSyncMode = useSyncModeModal();
  return (
    <Menu.Item itemKey="ethercat.syncMode" onClick={() => openSyncMode({ deviceKey })}>
      <Icon.Time />
      Set sync mode
    </Menu.Item>
  );
};

const TASK_CONTEXT_MENU_ITEMS = [
  Device.createTaskContextMenuItem({
    itemKey: "ethercat.readTask",
    label: "Create read task",
    useCreate: Task.useCreateRead,
  }),
  Device.createTaskContextMenuItem({
    itemKey: "ethercat.writeTask",
    label: "Create write task",
    useCreate: Task.useCreateWrite,
  }),
];

export const ContextMenuItems = (props: Device.ContextMenuItemsProps) => {
  const keys = props.selection.ids.map((id) => id.key);
  const hasUpdatePermission = Access.useUpdateGranted(device.ontologyID(keys));
  const { update: toggleEnabled } = useToggleEnabled();

  const { allDisabled, allEnabled } = useEnabledState({ keys });

  const handleDisable = () => {
    toggleEnabled({ keys, enabled: false });
  };

  const handleEnable = () => {
    toggleEnabled({ keys, enabled: true });
  };

  return (
    <>
      <Device.ConfigureMenuItem {...props} />
      <Device.ChangeIdentifierMenuItem {...props} icon="Logo.EtherCAT" />
      {FLAGS.ethercatSyncMode && hasUpdatePermission && keys.length === 1 && (
        <SyncModeItem deviceKey={keys[0]} />
      )}
      <Menu.Divider />
      <Device.TaskContextMenuItems {...props} items={TASK_CONTEXT_MENU_ITEMS} />
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
