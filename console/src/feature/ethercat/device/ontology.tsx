// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Icon, Menu } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import {
  useSelectEnabledState,
  useToggleEnabled,
} from "@/feature/ethercat/device/queries";
import { useConfigureModal } from "@/feature/ethercat/device/useConfigureModal";
import { Task } from "@/feature/ethercat/task";
import { Device } from "@/platform/device";
import { type Tree } from "@/platform/tree";

const TASK_CONTEXT_MENU_ITEM_CONFIGS: Device.TaskContextMenuItemConfig[] = [
  {
    itemKey: "ethercat.readTask",
    label: "Create read task",
    type: Task.READ_TYPE,
  },
  {
    itemKey: "ethercat.writeTask",
    label: "Create write task",
    type: Task.WRITE_TYPE,
  },
];

export const ContextMenuItems = (props: Tree.ContextMenuProps) => {
  const keys = props.selection.ids.map((id) => id.key);
  const { update: toggleEnabled } = useToggleEnabled();
  const configure = useConfigureModal();
  const onConfigure = (deviceKey: device.Key) => configure({ deviceKey });

  const { allDisabled, allEnabled } = useSelectEnabledState({ keys });

  const handleDisable = useCallback(() => {
    toggleEnabled({ keys, enabled: false });
  }, [keys, toggleEnabled]);

  const handleEnable = useCallback(() => {
    toggleEnabled({ keys, enabled: true });
  }, [keys, toggleEnabled]);

  return (
    <>
      <Device.ConfigureMenuItem {...props} onConfigure={onConfigure} />
      <Device.ChangeIdentifierMenuItem {...props} icon="Logo.EtherCAT" />
      <Menu.Divider />
      <Device.TaskContextMenuItems
        {...props}
        onConfigure={onConfigure}
        taskContextMenuItemConfigs={TASK_CONTEXT_MENU_ITEM_CONFIGS}
      />
      <Menu.Divider />
      {!allDisabled && (
        <Menu.Item itemKey="ethercat.disable" onClick={handleDisable}>
          <Icon.Disable />
          Disable
        </Menu.Item>
      )}
      {!allEnabled && (
        <Menu.Item itemKey="ethercat.enable" onClick={handleEnable}>
          <Icon.Enable />
          Enable
        </Menu.Item>
      )}
    </>
  );
};
