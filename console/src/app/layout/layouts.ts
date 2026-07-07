// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Main } from "@/app/main";
import { Mosaic } from "@/app/mosaic";
import { Selector } from "@/app/selector";
import { Vis } from "@/app/vis";
import { Arc } from "@/feature/arc";
import { Docs } from "@/feature/docs";
import { LinePlot } from "@/feature/lineplot";
import { Log } from "@/feature/log";
import { Range } from "@/feature/range";
import { Schematic } from "@/feature/schematic";
import { Status } from "@/feature/status";
import { Table } from "@/feature/table";
import { Task } from "@/feature/task";
import { type Layout } from "@/platform/layout";

export const LAYOUTS: Layout.Renderers = {
  ...Docs.LAYOUTS,
  ...Task.LAYOUTS,
  ...Main.LAYOUTS,
  ...Mosaic.LAYOUTS,
  ...Selector.LAYOUTS,
  ...LinePlot.LAYOUTS,
  ...Log.LAYOUTS,
  ...Range.LAYOUTS,
  ...Schematic.LAYOUTS,
  ...Table.LAYOUTS,
  ...Vis.LAYOUTS,
  ...Arc.LAYOUTS,
  ...Status.LAYOUTS,
};

export const CONTEXT_MENUS: Layout.ContextMenuRenderers = {
  ...Schematic.CONTEXT_MENUS,
  ...LinePlot.CONTEXT_MENUS,
};
