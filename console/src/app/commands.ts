// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Arc } from "@/feature/arc";
import { Channel } from "@/feature/channel";
import { Cluster } from "@/feature/cluster";
import { Device } from "@/feature/device";
import { Docs } from "@/feature/docs";
import { Import } from "@/feature/import";
import { Label } from "@/feature/label";
import { LinePlot } from "@/feature/lineplot";
import { Log } from "@/feature/log";
import { Persist } from "@/feature/persist";
import { Project } from "@/feature/project";
import { Range } from "@/feature/range";
import { Schematic } from "@/feature/schematic";
import { Status } from "@/feature/status";
import { Table } from "@/feature/table";
import { Task } from "@/feature/task";
import { User } from "@/feature/user";
import { Framer } from "@/primitive/framer";
import { type Palette } from "@/primitive/palette";
import { Theme } from "@/primitive/theme";

export const COMMANDS: Palette.Command[] = [
  ...Channel.COMMANDS,
  ...Cluster.COMMANDS,
  ...Device.COMMANDS,
  ...Docs.COMMANDS,
  ...Framer.COMMANDS,
  ...Import.COMMANDS,
  ...Label.COMMANDS,
  ...LinePlot.COMMANDS,
  ...Log.COMMANDS,
  ...Persist.COMMANDS,
  ...Range.COMMANDS,
  ...Schematic.COMMANDS,
  ...Table.COMMANDS,
  ...Task.COMMANDS,
  ...User.COMMANDS,
  ...Project.COMMANDS,
  ...Arc.COMMANDS,
  ...Status.COMMANDS,
  ...Theme.COMMANDS,
];
