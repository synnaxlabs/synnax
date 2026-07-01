// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Device } from "@/app/device";
import { Task } from "@/app/task";
import { Arc } from "@/feature/arc";
import { Channel } from "@/feature/channel";
import { LinePlot } from "@/feature/lineplot";
import { Log } from "@/feature/log";
import { Project } from "@/feature/project";
import { Range } from "@/feature/range";
import { Schematic } from "@/feature/schematic";
import { Table } from "@/feature/table";
import { Cluster } from "@/platform/cluster";
import { Link } from "@/platform/link";

export const useLinks = (): void => {
  const handlers = {
    arc: Arc.useLink(),
    channel: Channel.useLink(),
    device: Device.useLink(),
    lineplot: LinePlot.useLink(),
    log: Log.useLink(),
    range: Range.useLink(),
    schematic: Schematic.useLink(),
    table: Table.useLink(),
    task: Task.useLink(),
    project: Project.useLink(),
  };
  Link.useDeep(Cluster.useLink(), handlers);
};
