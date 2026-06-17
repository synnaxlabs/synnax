// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Export } from "@/export";
import { type Selector } from "@/selector";
import { Content } from "@/table/content/Content";
import { extract } from "@/table/export";
import { LAYOUT_TYPE } from "@/table/layout";
import { Selectable } from "@/table/Selectable";
import { type Tabs } from "@/panel/tabs/index";

export * from "@/table/content/Content";
export * from "@/table/export";
export * from "@/table/layout";
export * from "@/table/Selectable";
export * from "@/table/session/slice";
export * from "@/table/Toolbar";
export * from "@/table/useCreate";

export const EXTRACTORS: Export.Extractors = { [LAYOUT_TYPE]: extract };

export const TABS: Record<string, Tabs.Renderer> = { [LAYOUT_TYPE]: Content };

export const SELECTABLES: Selector.Selectable[] = [Selectable];
