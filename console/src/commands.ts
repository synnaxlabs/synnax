// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ChannelServices } from "@/channel/services";
import { Docs } from "@/docs";
import { Framer } from "@/framer";
import { Hardware } from "@/hardware";
import { Import } from "@/import";
import { LabelServices } from "@/label/services";
import { Service } from "@/layered/service";
import { NodeServices } from "@/node/services";
import { type Palette } from "@/palette";
import { Persist } from "@/persist";
import { ProjectServices } from "@/project/services";
import { RangeServices } from "@/range/services";
import { Status } from "@/status";
import { UserServices } from "@/user/services";

export const COMMANDS: Palette.Command[] = [
  ...ChannelServices.COMMANDS,
  ...NodeServices.COMMANDS,
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
