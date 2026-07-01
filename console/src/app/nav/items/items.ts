// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Nav } from "@/component/nav";
import { Arc } from "@/service/arc";
import { Channel } from "@/service/channel";
import { Device } from "@/service/device";
import { Project } from "@/service/project";
import { Range } from "@/service/range";
import { Status } from "@/service/status";
import { Task } from "@/service/task";
import { User } from "@/service/user";
import { Vis } from "@/service/vis";

export const DEFAULT_SIZE = 200;

export const LEFT: Nav.Item[] = [
  Channel.TOOLBAR,
  Range.TOOLBAR,
  Project.TOOLBAR,
  ...Task.NAV_DRAWER_ITEMS,
  ...Device.NAV_DRAWER_ITEMS,
  User.TOOLBAR,
  Arc.TOOLBAR,
  Status.TOOLBAR,
];

export const BOTTOM: Nav.Item = Vis.TOOLBAR;
