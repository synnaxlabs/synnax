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
import { Core } from "@/feature/core";
import { Device } from "@/feature/device";
import { LinePlot } from "@/feature/lineplot";
import { Link } from "@/feature/link";
import { Log } from "@/feature/log";
import { Project } from "@/feature/project";
import { Range } from "@/feature/range";
import { Schematic } from "@/feature/schematic";
import { Table } from "@/feature/table";
import { Task } from "@/feature/task";
import { Session } from "@/session";

// Keyed by the resource type, the segment after the cluster key in a deep link.
const useLinks = (): void =>
  Link.useDeep(Core.useLink(), {
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
  });

// Synnax Desktop takes one link, its login, and Account.Guard handles that one.
export const useDeep: () => void = Session.Runtime.LINKS_DISABLED ? () => {} : useLinks;
