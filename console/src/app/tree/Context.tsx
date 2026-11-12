// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement } from "react";

import { Access } from "@/feature/access";
import { Arc } from "@/feature/arc";
import { Channel } from "@/feature/channel";
import { Device } from "@/feature/device";
import { Group } from "@/feature/group";
import { LinePlot } from "@/feature/lineplot";
import { Log } from "@/feature/log";
import { Panel } from "@/feature/panel";
import { Project } from "@/feature/project";
import { Rack } from "@/feature/rack";
import { Schematic } from "@/feature/schematic";
import { Secret } from "@/feature/secret";
import { Table } from "@/feature/table";
import { Task } from "@/feature/task";
import { User } from "@/feature/user";
import { Group as PlatformGroup } from "@/platform/group";
import { Tree } from "@/platform/tree";

// The records that groups list, in order, wired here rather than by each feature.
const GROUP_MEMBERS: Group.Member[] = [
  Channel.GROUP_MEMBER,
  Rack.GROUP_MEMBER,
  Device.GROUP_MEMBER,
  Project.GROUP_MEMBER,
  Schematic.GROUP_MEMBER,
  LinePlot.GROUP_MEMBER,
  Log.GROUP_MEMBER,
  Table.GROUP_MEMBER,
  Panel.GROUP_MEMBER,
];

const TREE_ITEMS: Tree.Items = {
  ...(DESKTOP ? {} : Access.TREE_ITEMS),
  ...Arc.TREE_ITEMS,
  ...Channel.TREE_ITEMS,
  ...Device.TREE_ITEMS,
  ...Group.createTreeItems(GROUP_MEMBERS),
  ...LinePlot.TREE_ITEMS,
  ...Log.TREE_ITEMS,
  ...Panel.TREE_ITEMS,
  ...Project.TREE_ITEMS,
  ...Rack.TREE_ITEMS,
  ...Schematic.TREE_ITEMS,
  ...Secret.TREE_ITEMS,
  ...Table.TREE_ITEMS,
  ...Task.TREE_ITEMS,
  ...(DESKTOP ? {} : User.TREE_ITEMS),
};

export interface ContextProps extends PropsWithChildren<{}> {}

export const Context = ({ children }: ContextProps): ReactElement => (
  <Tree.Provider items={TREE_ITEMS} contextMenus={PlatformGroup.TREE_CONTEXT_MENUS}>
    {children}
  </Tree.Provider>
);
