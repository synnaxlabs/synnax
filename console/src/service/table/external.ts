// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Export } from "@/export";
import { extract } from "@/service/table/imex/export";
import { LAYOUT_TYPE } from "@/service/table/layout";
import { Selectable } from "@/service/table/Selectable";
import { Table } from "@/service/table/Table";
import { type Layout } from "@/layout";
import { type Selector } from "@/selector";

export * from "@/service/table/imex";
export * from "@/service/table/layout";
export * from "@/service/table/link";
export * from "@/service/table/ontology";
export * from "@/service/table/palette";
export * from "@/service/table/toolbar";
export * from "@/service/table/useCreate";

export const EXTRACTORS: Export.Extractors = { [LAYOUT_TYPE]: extract };

export const LAYOUTS: Record<string, Layout.Renderer> = { [LAYOUT_TYPE]: Table };

export const SELECTABLES: Selector.Selectable[] = [Selectable];
