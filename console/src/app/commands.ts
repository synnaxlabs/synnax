// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Framer } from "@/component/framer";
import { type Palette } from "@/component/palette";
import { Theme } from "@/component/theme";
import { Arc } from "@/service/arc";
import { Channel } from "@/service/channel";
import { Cluster } from "@/service/cluster";
import { Device } from "@/service/device";
import { Docs } from "@/service/docs";
import { Import } from "@/service/import";
import { Label } from "@/service/label";
import { LinePlot } from "@/service/lineplot";
import { Log } from "@/service/log";
import { Persist } from "@/service/persist";
import { Project } from "@/service/project";
import { Range } from "@/service/range";
import { Schematic } from "@/service/schematic";
import { Status } from "@/service/status";
import { Table } from "@/service/table";
import { Task } from "@/service/task";
import { User } from "@/service/user";

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
