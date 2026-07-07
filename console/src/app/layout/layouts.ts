// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Main, MAIN_LAYOUT_TYPE } from "@/app/Main";
import { Mosaic, MOSAIC_LAYOUT_TYPE, MosaicWindow } from "@/app/Mosaic";
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
import { Session } from "@/session";

export const LAYOUT_RENDERERS: Record<string, Layout.Renderer> = {
  ...Docs.LAYOUTS,
  ...Task.LAYOUTS,
  [MAIN_LAYOUT_TYPE]: Main,
  [MOSAIC_LAYOUT_TYPE]: Mosaic,
  [Session.Layout.MOSAIC_WINDOW_TYPE]: MosaicWindow,
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

export const CONTEXT_MENU_RENDERERS: Record<string, Layout.ContextMenuRenderer> = {
  ...Schematic.CONTEXT_MENUS,
  ...LinePlot.CONTEXT_MENUS,
};
