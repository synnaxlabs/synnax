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
import { Library } from "@/feature/library";
import { LinePlot } from "@/feature/lineplot";
import { Link } from "@/feature/link";
import { Log } from "@/feature/log";
import { Project } from "@/feature/project";
import { Range } from "@/feature/range";
import { Schematic } from "@/feature/schematic";
import { Table } from "@/feature/table";
import { Task } from "@/feature/task";
import { Session } from "@/session";

const LINKS: Link.Registry = {
  ...Arc.LINKS,
  ...Channel.LINKS,
  ...Device.LINKS,
  ...Library.LINKS,
  ...LinePlot.LINKS,
  ...Log.LINKS,
  ...Range.LINKS,
  ...Schematic.LINKS,
  ...Table.LINKS,
  ...Task.LINKS,
  ...Project.LINKS,
};

const useLinks = (): void => {
  const linkHandlers = Object.fromEntries(
    Object.entries(LINKS).map(([key, handler]) => [key, handler()]),
  );
  Link.useDeep(Core.useLink(), linkHandlers);
};

export const useDeep: () => void = Session.Runtime.LINKS_DISABLED ? () => {} : useLinks;
