// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { extract } from "@/feature/table/export";
import { ingest } from "@/feature/table/import";
import { Selectable } from "@/feature/table/Selectable";
import { Table } from "@/feature/table/Table";
import { type Export } from "@/platform/export";
import { type Import } from "@/platform/import";
import { type Layout } from "@/platform/layout";
import { type Selector } from "@/platform/selector";
import { Table as CommonTable } from "@/platform/table";

export * from "@/feature/table/import";
export * from "@/feature/table/link";
export * from "@/feature/table/ontology";
export * from "@/feature/table/palette";
export * from "@/feature/table/Toolbar";
export * from "@/platform/table/external";

export const EXTRACTORS: Export.Extractors = { [CommonTable.LAYOUT_TYPE]: extract };

export const FILE_INGESTERS: Import.FileIngesters = {
  [CommonTable.LAYOUT_TYPE]: ingest,
};

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [CommonTable.LAYOUT_TYPE]: Table,
};

export const SELECTABLES: Selector.Selectable[] = [Selectable];
