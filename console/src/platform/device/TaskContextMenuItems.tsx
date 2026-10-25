// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, task } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";
import { type FC } from "react";

import { Task } from "@/platform/task";
import { type Tree } from "@/platform/tree";

export interface TaskContextMenuItemConfig {
  itemKey: string;
  label: string;
  useCreate: Task.UseCreate;
}

export interface TaskContextMenuItemProps {
  deviceKey: device.Key;
  beforeCreate: () => void;
}

export interface TaskContextMenuItem extends FC<TaskContextMenuItemProps> {
  itemKey: string;
}

/** Builds the menu item that creates a task of the configured type for a device. */
export const createTaskContextMenuItem = ({
  itemKey,
  label,
  useCreate,
}: TaskContextMenuItemConfig): TaskContextMenuItem => {
  const Item: TaskContextMenuItem = ({ deviceKey, beforeCreate }) => {
    const createTask = useCreate();
    const handleClick = () => {
      beforeCreate();
      createTask({ deviceKey });
    };
    return (
      <Task.CreateMenuItem itemKey={itemKey} onClick={handleClick}>
        {label}
      </Task.CreateMenuItem>
    );
  };
  Item.itemKey = itemKey;
  return Item;
};

export interface TaskContextMenuItemsProps extends Pick<
  Tree.ContextMenuProps,
  "selection" | "state"
> {
  onConfigure: (deviceKey: device.Key) => void;
  items: TaskContextMenuItem[];
}

export const TaskContextMenuItems = ({
  onConfigure,
  state: { getResource },
  selection: { ids },
  items,
}: TaskContextMenuItemsProps) => {
  const hasCreatePermission = Access.useCreateGranted(task.TYPE_ONTOLOGY_ID);
  const firstID = ids[0];
  const maybeConfigure = () => {
    if (getResource(firstID).data?.configured !== true) onConfigure(firstID.key);
  };
  if (!hasCreatePermission) return null;
  return (
    <>
      {items.map((Item) => (
        <Item
          key={Item.itemKey}
          deviceKey={firstID.key}
          beforeCreate={maybeConfigure}
        />
      ))}
    </>
  );
};
