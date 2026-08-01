// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table } from "@synnaxlabs/client";
import { Icon, Table as Base } from "@synnaxlabs/pluto";

import { Selectable } from "@/feature/table/Selectable";
import { Table } from "@/feature/table/Table";
import { Toolbar } from "@/feature/table/Toolbar";
import { Panel } from "@/platform/panel";
import { type Selector } from "@/platform/selector";

export * from "@/feature/table/commands";
export * from "@/feature/table/link";
export * from "@/feature/table/search";
export * from "@/feature/table/Toolbar";
export * from "@/feature/table/tree";
export * from "@/platform/table/external";

const TAB_TYPE = table.TYPE_ONTOLOGY_ID.type;

export const SELECTABLES: Selector.Selectable[] = [Selectable];

const TAB: Panel.Tab = {
  Content: Table,
  Toolbar,
  Name: Panel.createEditableTabName(Base, <Icon.Table />),
};

export const TABS: Panel.Tabs = {
  [TAB_TYPE]: TAB,
};
