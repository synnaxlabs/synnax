// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Selector } from "@/app/selector";
import { Arc } from "@/feature/arc";
import { Docs } from "@/feature/docs";
import { LinePlot } from "@/feature/lineplot";
import { Log } from "@/feature/log";
import { Range } from "@/feature/range";
import { Schematic } from "@/feature/schematic";
import { Status } from "@/feature/status";
import { Table } from "@/feature/table";
import { Task } from "@/feature/task";
import { type Panel } from "@/platform/panel";

export const TABS: Panel.Tabs = {
  ...Status.TABS,
  ...Arc.TABS,
  ...Range.TABS,
  ...Docs.TABS,
  ...Selector.TABS,
  ...LinePlot.TABS,
  ...Log.TABS,
  ...Schematic.TABS,
  ...Table.TABS,
  ...Task.TABS,
};
