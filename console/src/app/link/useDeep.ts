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
import { LinePlot } from "@/feature/lineplot";
import { Link } from "@/feature/link";
import { Log } from "@/feature/log";
import { Project } from "@/feature/project";
import { Range } from "@/feature/range";
import { Schematic } from "@/feature/schematic";
import { Table } from "@/feature/table";
import { Task } from "@/feature/task";

const LINKS: Link.Registry = {
  ...Arc.LINKS,
  ...Channel.LINKS,
  ...Device.LINKS,
  ...LinePlot.LINKS,
  ...Log.LINKS,
  ...Range.LINKS,
  ...Schematic.LINKS,
  ...Table.LINKS,
  ...Task.LINKS,
  ...Project.LINKS,
};

export const useDeep = () => {
  const linkHandlers = Object.fromEntries(
    Object.entries(LINKS).map(([key, handler]) => [key, handler()]),
  );
  Link.useDeep(Cluster.useLink(), linkHandlers);
};
