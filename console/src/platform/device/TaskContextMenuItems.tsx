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

import { type Ontology } from "@/platform/ontology";
import { Panel } from "@/platform/panel";
import { Task } from "@/platform/task";

export interface TaskContextMenuItemConfig {
  itemKey: string;
  label: string;
  type: string;
}

export interface TaskContextMenuItemsProps extends Pick<
  Ontology.TreeContextMenuProps,
  "selection" | "state"
> {
  onConfigure: (deviceKey: device.Key) => void;
  taskContextMenuItemConfigs: TaskContextMenuItemConfig[];
}

export const TaskContextMenuItems = ({
  onConfigure,
  state: { getResource },
  selection: { ids },
  taskContextMenuItemConfigs,
}: TaskContextMenuItemsProps) => {
  const openTab = Panel.useOpenTab();
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
      {taskContextMenuItemConfigs.map(({ itemKey, label, type }) => {
        const handleClick = () => {
          maybeConfigure();
          openTab({ variant: "view", type, args: { deviceKey: key } });
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
