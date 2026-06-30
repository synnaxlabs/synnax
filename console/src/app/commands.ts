// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ClusterServices } from "@/cluster/services";
import { Import } from "@/import";
import { type Palette } from "@/palette";
import { ProjectServices } from "@/project/services";
import { RangeServices } from "@/range/services";
import { Service } from "@/service";
import { ChannelServices } from "@/service/channel";
import { Docs } from "@/service/docs";
import { Framer } from "@/service/framer";
import { Hardware } from "@/service/hardware";
import { LabelServices } from "@/service/label/services";
import { Status } from "@/service/status";
import { UserServices } from "@/service/user/services";
import { Persist } from "@/session/persist";

export const COMMANDS: Palette.Command[] = [
  ...ChannelServices.COMMANDS,
  ...ClusterServices.COMMANDS,
  ...Docs.COMMANDS,
  ...Framer.COMMANDS,
  ...Hardware.COMMANDS,
  ...Import.COMMANDS,
  ...LabelServices.COMMANDS,
  ...Service.LinePlot.COMMANDS,
  ...Service.Log.COMMANDS,
  ...Persist.COMMANDS,
  ...RangeServices.COMMANDS,
  ...Service.Schematic.COMMANDS,
  ...Service.Table.COMMANDS,
  ...UserServices.COMMANDS,
  ...ProjectServices.COMMANDS,
  ...Service.Arc.COMMANDS,
  ...Status.COMMANDS,
  ...Service.Theme.COMMANDS,
];
