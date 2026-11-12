// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement } from "react";

import { Arc } from "@/feature/arc";
import { Channel } from "@/feature/channel";
import { Device } from "@/feature/device";
import { LinePlot } from "@/feature/lineplot";
import { Log } from "@/feature/log";
import { Project } from "@/feature/project";
import { Range } from "@/feature/range";
import { Schematic } from "@/feature/schematic";
import { Table } from "@/feature/table";
import { Task } from "@/feature/task";
import { Link } from "@/platform/link";

/** Gives the window the handler that opens each type of resource. */
export const Provider = ({ children }: PropsWithChildren): ReactElement => (
  <Link.Provider
    // Keyed by the resource type, the segment after the cluster key in a deep link.
    handlers={{
      arc: Arc.useLink(),
      channel: Channel.useLink(),
      device: Device.useLink(),
      lineplot: LinePlot.useLink(),
      log: Log.useLink(),
      project: Project.useLink(),
      range: Range.useLink(),
      schematic: Schematic.useLink(),
      table: Table.useLink(),
      task: Task.useLink(),
    }}
  >
    {children}
  </Link.Provider>
);
