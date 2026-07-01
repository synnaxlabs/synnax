// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { extract } from "@/feature/table/export";
import { FILE_INGESTERS } from "@/feature/table/import";
import { Selectable } from "@/feature/table/Selectable";
import { Table } from "@/feature/table/Table";
import { type Export } from "@/platform/export";
import { type Layout } from "@/platform/layout";
import { type Selector } from "@/platform/selector";
import { LAYOUT_TYPE } from "@/platform/table/layout";

export * from "@/feature/table/link";
export * from "@/feature/table/ontology";
export * from "@/feature/table/palette";
export * from "@/feature/table/Toolbar";
export * from "@/platform/table/layout";

export const EXTRACTORS: Export.Extractors = { [LAYOUT_TYPE]: extract };

export const ImEx = { FILE_INGESTERS };

export const LAYOUTS: Record<string, Layout.Renderer> = { [LAYOUT_TYPE]: Table };

export const SELECTABLES: Selector.Selectable[] = [Selectable];
export * from "@/platform/table/useCreate";
