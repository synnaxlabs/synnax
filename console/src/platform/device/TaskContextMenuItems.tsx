// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, query, task } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { type FC } from "react";

import { useConfigureMany } from "@/platform/device/identifier";
import { Task } from "@/platform/task";
import { type Tree } from "@/platform/tree";

export interface TaskContextMenuItemConfig {
  itemKey: string;
  label: string;
  useCreate: Task.UseCreate;
  /**
   * Offers the item for several devices, which it configures with these zero properties
   * and a free identifier when they are not configured yet. The task it creates gets
   * `deviceKeys`.
   */
  multiDevice?: { zeroProperties: record.Unknown };
}

export interface TaskContextMenuItemProps {
  deviceKeys: device.Key[];
  /** Runs before the item creates a task for one device. */
  beforeCreate: () => void;
}

export interface TaskContextMenuItem extends FC<TaskContextMenuItemProps> {
  itemKey: string;
  multiDevice: boolean;
}

/** Builds the menu item that creates a task of the configured type for devices. */
export const createTaskContextMenuItem = ({
  itemKey,
  label,
  useCreate,
  multiDevice,
}: TaskContextMenuItemConfig): TaskContextMenuItem => {
  const Item: TaskContextMenuItem = ({ deviceKeys, beforeCreate }) => {
    const createTask = useCreate();
    const { updateAsync: configure } = useConfigureMany();
    const handleClick = () => {
      if (multiDevice == null || deviceKeys.length === 1) {
        beforeCreate();
        createTask({ deviceKey: deviceKeys[0] });
        return;
      }
      const { zeroProperties } = multiDevice;
      void (async () => {
        if (await configure({ keys: deviceKeys, zeroProperties }))
          createTask({ deviceKeys });
      })();
    };
    return (
      <Task.CreateMenuItem itemKey={itemKey} onClick={handleClick}>
        {label}
      </Task.CreateMenuItem>
    );
  };
  Item.itemKey = itemKey;
  Item.multiDevice = multiDevice != null;
  return Item;
};

export interface TaskContextMenuItemsProps extends Pick<
  Tree.ContextMenuProps,
  "client" | "selection"
> {
  onConfigure: (deviceKey: device.Key) => void;
  items: TaskContextMenuItem[];
}

export const TaskContextMenuItems = ({
  client,
  onConfigure,
  selection: { ids },
  items,
}: TaskContextMenuItemsProps) => {
  const hasCreatePermission = Access.useCreateGranted(task.TYPE_RESOURCE_ID);
  const deviceKeys = ids.map(({ key }) => key);
  const shown = deviceKeys.length === 1 ? items : items.filter((i) => i.multiDevice);
  if (!hasCreatePermission || shown.length === 0) return null;
  const maybeConfigure = () => {
    const dev = client.devices.watch(deviceKeys[0]).getCached();
    if (!query.isLive(dev) || !dev.configured) onConfigure(deviceKeys[0]);
  };
  return (
    <>
      {shown.map((Item) => (
        <Item
          key={Item.itemKey}
          deviceKeys={deviceKeys}
          beforeCreate={maybeConfigure}
        />
      ))}
    </>
  );
};
