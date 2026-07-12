// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Access } from "@/feature/access";
import { Arc } from "@/feature/arc";
import { Channel } from "@/feature/channel";
import { Cluster } from "@/feature/cluster";
import { Device } from "@/feature/device";
import { Group } from "@/feature/group";
import { Label } from "@/feature/label";
import { LinePlot } from "@/feature/lineplot";
import { Log } from "@/feature/log";
import { Panel } from "@/feature/panel";
import { Project } from "@/feature/project";
import { Rack } from "@/feature/rack";
import { Range } from "@/feature/range";
import { Schematic } from "@/feature/schematic";
import { Status } from "@/feature/status";
import { Table } from "@/feature/table";
import { Task } from "@/feature/task";
import { User } from "@/feature/user";
import { type Tree } from "@/platform/tree";

export const TREE_ITEMS: Tree.Items = {
  ...Access.TREE_ITEMS,
  ...Arc.TREE_ITEMS,
  ...Channel.TREE_ITEMS,
  ...Cluster.TREE_ITEMS,
  ...Device.TREE_ITEMS,
  ...Group.TREE_ITEMS,
  ...Label.TREE_ITEMS,
  ...LinePlot.TREE_ITEMS,
  ...Log.TREE_ITEMS,
  ...Panel.TREE_ITEMS,
  ...Project.TREE_ITEMS,
  ...Rack.TREE_ITEMS,
  ...Range.TREE_ITEMS,
  ...Schematic.TREE_ITEMS,
  ...Status.TREE_ITEMS,
  ...Table.TREE_ITEMS,
  ...Task.TREE_ITEMS,
  ...User.TREE_ITEMS,
};
