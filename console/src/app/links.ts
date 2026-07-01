// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Project } from "@/service/project";
import { Range } from "@/service/range";
import { Arc } from "@/service/arc";
import { Channel } from "@/service/channel";
import { Cluster } from "@/service/cluster";
import { Device } from "@/service/device";
import { LinePlot } from "@/service/lineplot";
import { Link } from "@/service/link";
import { Log } from "@/service/log";
import { Schematic } from "@/service/schematic";
import { Table } from "@/service/table";
import { Task } from "@/service/task";

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
