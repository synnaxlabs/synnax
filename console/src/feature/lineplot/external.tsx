// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { Icon, LinePlot as Base } from "@synnaxlabs/pluto";

import { LinePlot } from "@/feature/lineplot/LinePlot";
import { Selectable } from "@/feature/lineplot/Selectable";
import { Toolbar } from "@/feature/lineplot/toolbar";
import { Panel } from "@/platform/panel";
import { type Selector } from "@/platform/selector";

export * from "@/feature/lineplot/commands";
export * from "@/feature/lineplot/LinePlot";
export * from "@/feature/lineplot/link";
export * from "@/feature/lineplot/search";
export * from "@/feature/lineplot/toolbar";
export * from "@/feature/lineplot/tree";
export * from "@/feature/lineplot/useTriggerHold";
export * from "@/platform/lineplot/external";

const TAB_TYPE = lineplot.TYPE_ONTOLOGY_ID.type;

export const SELECTABLES: Selector.Selectable[] = [Selectable];

const TAB: Panel.Tab = {
  Content: LinePlot,
  Toolbar,
  Name: Panel.createEditableTabName(Base, <Icon.LinePlot />),
};

export const TABS: Panel.Tabs = {
  [TAB_TYPE]: TAB,
};
