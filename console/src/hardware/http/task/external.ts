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
import { Scan, SCAN_LAYOUT, ScanSelectable } from "@/hardware/http/task/Scan";
import { SCAN_TYPE } from "@/hardware/http/task/types";
import { type Layout } from "@/layout";
import { type Selector } from "@/selector";

export * from "@/hardware/http/task/palette";
export * from "@/hardware/http/task/Scan";
export * from "@/hardware/http/task/types";

export const EXTRACTORS: Export.Extractors = {
  [SCAN_TYPE]: Common.Task.extract,
};

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [SCAN_TYPE]: Scan,
};

export const SELECTABLES: Selector.Selectable[] = [ScanSelectable];

export const ZERO_LAYOUTS: Record<string, Common.Task.Layout> = {
  [SCAN_TYPE]: SCAN_LAYOUT,
};
