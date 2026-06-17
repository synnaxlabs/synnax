// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Export } from "@/export";
import { Content } from "@/schematic/content/Content";
import { extract } from "@/schematic/imex";
import { Selectable } from "@/schematic/Selectable";
import { Tab } from "@/schematic/tab";
import { type Selector } from "@/selector";
import { type Tabs } from "@/panel/tabs/index";

export * from "@/schematic/content/Content";
export * from "@/schematic/session/slice";
export * from "@/schematic/toolbar";

export const EXTRACTORS: Export.Extractors = { [Tab.TYPE]: extract };

export const TABS: Record<string, Tabs.Renderer> = {
  [Tab.TYPE]: Content,
};

export const SELECTABLES: Selector.Selectable[] = [Selectable];
