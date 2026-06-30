// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ClusterServices } from "@/cluster/services";
import { ProjectServices } from "@/project/services";
import { RangeServices } from "@/range/services";
import { Arc } from "@/service/arc";
import { ChannelServices } from "@/service/channel";
import { Hardware } from "@/service/hardware";
import { LinePlot } from "@/service/lineplot";
import { Link } from "@/service/link";
import { Log } from "@/service/log";
import { Schematic } from "@/service/schematic";
import { Table } from "@/service/table";

export const useLinks = (): void => {
  const handlers = {
    ...Hardware.useLinks(),
    arc: Arc.useLink(),
    channel: ChannelServices.useLink(),
    lineplot: LinePlot.useLink(),
    log: Log.useLink(),
    range: RangeServices.useLink(),
    schematic: Schematic.useLink(),
    table: Table.useLink(),
    project: ProjectServices.useLink(),
  };
  Link.useDeep(ClusterServices.useLink(), handlers);
};
