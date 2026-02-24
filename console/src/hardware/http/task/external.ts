// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Export } from "@/export";
import { Common } from "@/hardware/common";
import { Read, READ_LAYOUT, ReadSelectable } from "@/hardware/http/task/Read";
import { READ_TYPE, readConfigZ } from "@/hardware/http/task/types";
import { type Import } from "@/import";
import { type Layout } from "@/layout";
import { type Selector } from "@/selector";

export * from "@/hardware/http/task/palette";
export * from "@/hardware/http/task/Read";
export * from "@/hardware/http/task/types";

export const EXTRACTORS: Export.Extractors = {
  [READ_TYPE]: Common.Task.extract,
};

export const FILE_INGESTERS: Import.FileIngesters = {
  [READ_TYPE]: Common.Task.createIngester(readConfigZ, READ_LAYOUT),
};

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [READ_TYPE]: Read,
};

export const SELECTABLES: Selector.Selectable[] = [ReadSelectable];

export const ZERO_LAYOUTS: Record<string, Common.Task.Layout> = {
  [READ_TYPE]: READ_LAYOUT,
};
