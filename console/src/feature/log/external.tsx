// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { Icon, Log as Base } from "@synnaxlabs/pluto";

import { Log } from "@/feature/log/Log";
import { Selectable } from "@/feature/log/Selectable";
import { Toolbar } from "@/feature/log/toolbar";
import { Panel } from "@/platform/panel";
import { type Selector } from "@/platform/selector";

export * from "@/feature/log/commands";
export * from "@/feature/log/link";
export * from "@/feature/log/search";
export * from "@/feature/log/Selectable";
export * from "@/feature/log/toolbar";
export * from "@/feature/log/tree";
export * from "@/platform/log/external";

const TAB_TYPE = log.TYPE_ONTOLOGY_ID.type;

export const SELECTABLES: Selector.Selectable[] = [Selectable];

const TAB: Panel.Tab = {
  Content: Log,
  Toolbar,
  Name: Panel.createEditableTabName(Base, <Icon.Log />),
};

export const TABS: Panel.Tabs = {
  [TAB_TYPE]: TAB,
};
