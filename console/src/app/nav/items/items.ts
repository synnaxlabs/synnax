// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Task } from "@/app/task";
import { Vis } from "@/app/vis";
import { Arc } from "@/feature/arc";
import { Channel } from "@/feature/channel";
import { Device } from "@/feature/device";
import { Project } from "@/feature/project";
import { Range } from "@/feature/range";
import { Status } from "@/feature/status";
import { User } from "@/feature/user";
import { type Nav } from "@/platform/nav";

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
