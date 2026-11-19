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
import { strings, unique } from "@synnaxlabs/x";

import {
  useEnabledState,
  useSlavesByKeys,
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

const CONFIGURE_MANY = {
  zeroProperties: {},
  suggestIdentifiers: ({ name }: device.Device) => strings.createShortIdentifiers(name),
};

const READ_TASK_ITEM = Device.createTaskContextMenuItem({
  itemKey: "ethercat.readTask",
  label: "Create read task",
  useCreate: Task.useCreateRead,
  configureMany: CONFIGURE_MANY,
});

const WRITE_TASK_ITEM = Device.createTaskContextMenuItem({
  itemKey: "ethercat.writeTask",
  label: "Create write task",
  useCreate: Task.useCreateWrite,
  configureMany: CONFIGURE_MANY,
});

export const ContextMenuItems = (props: Device.ContextMenuItemsProps) => {
  const keys = props.selection.ids.map((id) => id.key);
  const hasUpdatePermission = Access.useUpdateGranted(device.resourceID(keys));
  const { update: toggleEnabled } = useToggleEnabled();

  const { allDisabled, allEnabled } = useEnabledState({ keys });
  const slaves = useSlavesByKeys(keys) ?? [];
  // A task takes only the slaves with entries on its side, all on one network.
  const selectionOf = (side: Task.PDOType) => {
    const used = slaves.filter(({ properties }) => properties.pdos[side].length > 0);
    if (unique.unique(used.map(({ properties }) => properties.network)).length !== 1)
      return null;
    const usedKeys = new Set(used.map(({ key }) => key));
    return {
      ...props.selection,
      ids: props.selection.ids.filter(({ key }) => usedKeys.has(key)),
    };
  };
  const taskItems = (
    [
      [READ_TASK_ITEM, "inputs"],
      [WRITE_TASK_ITEM, "outputs"],
    ] as const
  ).flatMap(([item, side]) => {
    const selection = selectionOf(side);
    return selection == null ? [] : [{ item, selection }];
  });

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
          {taskItems.map(({ item, selection }) => (
            <Device.TaskContextMenuItems
              key={item.itemKey}
              {...props}
              selection={selection}
              items={[item]}
            />
          ))}
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
