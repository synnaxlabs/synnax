// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { Access } from "@/feature/access";
import { Arc } from "@/feature/arc";
import { Channel } from "@/feature/channel";
import { Device } from "@/feature/device";
import { Group } from "@/feature/group";
import { LinePlot } from "@/feature/lineplot";
import { Log } from "@/feature/log";
import { Project } from "@/feature/project";
import { Rack } from "@/feature/rack";
import { Range } from "@/feature/range";
import { Schematic } from "@/feature/schematic";
import { Table } from "@/feature/table";
import { Task } from "@/feature/task";
import { User } from "@/feature/user";
import { Tree } from "@/platform/tree";

// Resource types with no registered Item (builtin, range-alias, framer, panel, view)
// fall back to Tree.DEFAULT_ITEM. Icon-only types get a bare item carrying just an icon.
export const TREE_ITEMS: Tree.Items = {
  schematic: Schematic.TREE_ITEM,
  user: User.TREE_ITEM,
  group: Group.TREE_ITEM,
  range: Range.TREE_ITEM,
  project: Project.TREE_ITEM,
  lineplot: LinePlot.TREE_ITEM,
  rack: Rack.TREE_ITEM,
  task: Task.TREE_ITEM,
  device: Device.TREE_ITEM,
  channel: Channel.TREE_ITEM,
  log: Log.TREE_ITEM,
  table: Table.TREE_ITEM,
  arc: Arc.TREE_ITEM,
  policy: Access.Policy.TREE_ITEM,
  role: Access.Role.TREE_ITEM,
  schematic_symbol: Tree.createItem({
    type: "schematic_symbol",
    icon: <Icon.Schematic />,
  }),
  node: Tree.createItem({ type: "node", icon: <Icon.Node /> }),
  label: Tree.createItem({ type: "label", icon: <Icon.Label /> }),
  status: Tree.createItem({ type: "status", icon: <Icon.Status /> }),
};
