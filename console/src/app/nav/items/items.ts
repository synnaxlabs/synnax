// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ProjectServices } from "@/project/services";
import { Range } from "@/range";
import { type Service } from "@/service";
import { Arc } from "@/service/arc";
import { ChannelServices } from "@/service/channel";
import { Hardware } from "@/service/hardware";
import { UserServices } from "@/service/user/services";
import { Status } from "@/status";
import { Vis } from "@/vis";

export const DEFAULT_SIZE = 200;

export const LEFT: Service.Nav.Item[] = [
  ChannelServices.TOOLBAR,
  Range.TOOLBAR,
  ProjectServices.TOOLBAR,
  ...Hardware.NAV_DRAWER_ITEMS,
  UserServices.TOOLBAR,
  Arc.TOOLBAR,
  Status.TOOLBAR,
];

export const BOTTOM: Service.Nav.Item = Vis.TOOLBAR;
