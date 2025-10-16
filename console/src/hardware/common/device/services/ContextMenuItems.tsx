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
  layout: Task.Layout;
}

export interface ContextMenuItemsProps
  extends Pick<Ontology.TreeContextMenuProps, "selection" | "state"> {
  children?: ReactElement;
  configureLayout: Layout.BaseState;
  taskContextMenuItemConfigs: TaskContextMenuItemConfig[];
}

export const ContextMenuItems = ({
  children,
  configureLayout,
  state: { getResource },
  selection: { ids },
  taskContextMenuItemConfigs,
}: ContextMenuItemsProps) => {
  const placeLayout = Layout.usePlacer();
  const firstID = ids[0];
  const first = getResource(firstID);
  const key = first.id.key;
  const maybeConfigure = () => {
    if (first.data?.configured !== true) placeLayout({ ...configureLayout, key });
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
          <Task.CreateContextMenuItem key={itemKey} onClick={handleClick}>
            {label}
          </Task.CreateContextMenuItem>
        );
      })}
    </>
  );
};
