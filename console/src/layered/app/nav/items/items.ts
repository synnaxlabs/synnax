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
import { type Service } from "@/layered/service";
import { ProjectServices } from "@/project/services";
import { Range } from "@/range";
import { Status } from "@/status";
import { UserServices } from "@/user/services";
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
