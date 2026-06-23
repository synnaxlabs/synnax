// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Arc } from "@/arc";
import { ChannelServices } from "@/channel/services";
import { Hardware } from "@/hardware";
import { type View } from "@/layered/view";
import { ProjectServices } from "@/project/services";
import { Range } from "@/range";
import { Status } from "@/status";
import { UserServices } from "@/user/services";
import { Vis } from "@/vis";

export const LEFT: View.Nav.Item[] = [
  ...Hardware.NAV_DRAWER_ITEMS,
  Arc.TOOLBAR,
  Range.TOOLBAR,
  Status.TOOLBAR,
  ChannelServices.TOOLBAR,
  ProjectServices.TOOLBAR,
  UserServices.TOOLBAR,
];

export const BOTTOM: View.Nav.Item = Vis.TOOLBAR;
