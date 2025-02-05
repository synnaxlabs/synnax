// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Task } from "@/hardware/common/task";
import { Layout } from "@/layout";
import { type Ontology } from "@/ontology";

export interface TaskContextMenuItemConfig {
  itemKey: string;
  label: string;
  layout: Layout.BaseState;
}

export interface ContextMenuItemsProps {
  selection: Ontology.TreeContextMenuProps["selection"];
  deviceConfigLayout: Omit<Layout.BaseState, "key">;
  taskContextMenuItemConfigs: TaskContextMenuItemConfig[];
  children?: ReactElement;
}

export const ContextMenuItems = ({
  selection: { resources },
  deviceConfigLayout,
  taskContextMenuItemConfigs,
  children,
}: ContextMenuItemsProps): ReactElement | null => {
  const placeLayout = Layout.usePlacer();
  if (resources.length !== 1) return null;
  const key = resources[0].id.key;
  const maybeConfigure = () => {
    if (resources[0].data?.configured !== true)
      placeLayout({ ...deviceConfigLayout, key });
  };
  return (
    <>
      {children}
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
