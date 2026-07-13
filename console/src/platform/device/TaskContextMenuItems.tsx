// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, task } from "@synnaxlabs/client";
import { Access, Device } from "@synnaxlabs/pluto";

import { Layout } from "@/platform/layout";
import { Task } from "@/platform/task";
import { type Tree } from "@/platform/tree";

export interface TaskContextMenuItemConfig {
  itemKey: string;
  label: string;
  layout: Task.Layout;
}

export interface TaskContextMenuItemsProps extends Pick<
  Tree.ContextMenuProps,
  "selection"
> {
  onConfigure: (deviceKey: device.Key) => void;
  taskContextMenuItemConfigs: TaskContextMenuItemConfig[];
}

export const TaskContextMenuItems = ({
  onConfigure,
  selection: { ids },
  taskContextMenuItemConfigs,
}: TaskContextMenuItemsProps) => {
  const placeLayout = Layout.usePlacer();
  const hasCreatePermission = Access.useCreateGranted(task.TYPE_ONTOLOGY_ID);
  const key = ids[0].key;
  const configured = Device.useRetrieve({ key }).data?.configured;
  const maybeConfigure = () => {
    if (configured !== true) onConfigure(key);
  };
  if (!hasCreatePermission) return null;
  return (
    <>
      {taskContextMenuItemConfigs.map(({ itemKey, label, layout }) => {
        const handleClick = () => {
          maybeConfigure();
          placeLayout({ ...layout, args: { deviceKey: key } });
        };
        return (
          <Task.CreateMenuItem key={itemKey} itemKey={itemKey} onClick={handleClick}>
            {label}
          </Task.CreateMenuItem>
        );
      })}
    </>
  );
};
