// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";

import { Task } from "@/hardware/common/task";
import { Layout } from "@/layout";
import { type Ontology } from "@/ontology";

export interface TaskContextMenuItemConfig {
  itemKey: string;
  label: string;
  layout: Task.Layout;
}

export interface TaskContextMenuItemsProps extends Pick<
  Ontology.TreeContextMenuProps,
  "selection" | "state"
> {
  configureLayout: Layout.BaseState;
  taskContextMenuItemConfigs: TaskContextMenuItemConfig[];
}

export const TaskContextMenuItems = ({
  configureLayout,
  state: { getResource },
  selection: { ids },
  taskContextMenuItemConfigs,
}: TaskContextMenuItemsProps) => {
  const placeLayout = Layout.usePlacer();
  const hasCreatePermission = Access.useCreateGranted(task.TYPE_ONTOLOGY_ID);
  const firstID = ids[0];
  const first = getResource(firstID);
  const key = first.id.key;
  const maybeConfigure = () => {
    if (first.data?.configured !== true) placeLayout({ ...configureLayout, key });
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
