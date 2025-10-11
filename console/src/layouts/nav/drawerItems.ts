// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ChannelServices } from "@/channel/services";
import { Hardware } from "@/hardware";
import { type Layout } from "@/layout";
import { Range } from "@/range";
import { UserServices } from "@/user/services";
import { Vis } from "@/vis";
import { WorkspaceServices } from "@/workspace/services";

export const DRAWER_ITEMS: Layout.NavDrawerItem[] = [
  ...Hardware.NAV_DRAWER_ITEMS,
  Range.TOOLBAR,
  Vis.TOOLBAR,
  ChannelServices.TOOLBAR,
  WorkspaceServices.TOOLBAR,
  UserServices.TOOLBAR,
];
