// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, query, task } from "@synnaxlabs/client";
import { Access, Device } from "@synnaxlabs/pluto";
import { type FC } from "react";

import {
  type ConfigureManyParams,
  useConfigureMany,
} from "@/platform/device/useConfigureMany";
import { Task } from "@/platform/task";
import { type Tree } from "@/platform/tree";

export interface TaskContextMenuItemConfig {
  itemKey: string;
  label: string;
  useCreate: Task.UseCreate;
  /**
   * Offers the item for several devices. It configures each one that is not configured
   * yet, then creates one task with `deviceKeys`.
   */
  configureMany?: Omit<ConfigureManyParams, "keys">;
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
  configureMany,
}: TaskContextMenuItemConfig): TaskContextMenuItem => {
  const Item: TaskContextMenuItem = ({ deviceKeys, beforeCreate }) => {
    const createTask = useCreate();
    const configure = useConfigureMany();
    const handleClick = () => {
      if (configureMany == null || deviceKeys.length === 1) {
        beforeCreate();
        createTask({ deviceKey: deviceKeys[0] });
        return;
      }
      void (async () => {
        if (await configure({ ...configureMany, keys: deviceKeys }))
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
  Item.multiDevice = configureMany != null;
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
  const several = deviceKeys.length > 1;
  const { data: devices } = Device.useResultMultiple(
    several ? { where: { key: { in: deviceKeys } } } : null,
  );
  // A task runs on one rack.
  const oneRack =
    devices?.length === deviceKeys.length &&
    new Set(devices.map(({ rack }) => rack)).size === 1;
  let shown = items;
  if (several) shown = oneRack ? items.filter((i) => i.multiDevice) : [];
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
