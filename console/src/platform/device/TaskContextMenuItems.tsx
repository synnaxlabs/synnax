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

import { Task } from "@/platform/task";
import { type Tree } from "@/platform/tree";

export interface TaskContextMenuItemConfig {
  itemKey: string;
  label: string;
  useCreate: Task.UseCreate;
}

export interface TaskContextMenuItemsProps extends Pick<
  Tree.ContextMenuProps,
  "selection" | "state"
> {
  onConfigure: (deviceKey: device.Key) => void;
  taskContextMenuItemConfigs: TaskContextMenuItemConfig[];
}

interface ItemProps extends TaskContextMenuItemConfig {
  deviceKey: device.Key;
  beforeCreate: () => void;
}

const Item = ({ itemKey, label, useCreate, deviceKey, beforeCreate }: ItemProps) => {
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

export const TaskContextMenuItems = ({
  onConfigure,
  state: { getResource },
  selection: { ids },
  taskContextMenuItemConfigs,
}: TaskContextMenuItemsProps) => {
  const hasCreatePermission = Access.useCreateGranted(task.TYPE_ONTOLOGY_ID);
  const firstID = ids[0];
  const first = getResource(firstID);
  const key = first.id.key;
  const maybeConfigure = () => {
    if (first.data?.configured !== true) onConfigure(key);
  };
  if (!hasCreatePermission) return null;
  return (
    <>
      {taskContextMenuItemConfigs.map((config) => (
        <Item
          key={config.itemKey}
          {...config}
          deviceKey={key}
          beforeCreate={maybeConfigure}
        />
      ))}
    </>
  );
};
