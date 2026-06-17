// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";

import { type Export } from "@/export";
import { extract } from "@/lineplot/export";
import { Selectable } from "@/lineplot/Selectable";
import { type Selector } from "@/selector";
import { type Tabs } from "@/tabs";

export * from "@/lineplot/addChannelsToActivePlot";
export * from "@/lineplot/Controls";
export * from "@/lineplot/export";
export * from "@/lineplot/LinePlot";
export * from "@/lineplot/toolbar";
export * from "@/lineplot/useCreate";
export * from "@/lineplot/useTriggerHold";

const TYPE: ontology.ResourceType = "lineplot";

export const EXTRACTORS: Export.Extractors = { [TYPE]: extract };

export const TABS: Record<string, Tabs.Renderer> = { [TYPE]: LinePlot };

export const SELECTABLES: Selector.Selectable[] = [Selectable];
