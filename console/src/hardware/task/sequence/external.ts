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
import { ingest } from "@/hardware/task/sequence/import";
import { LAYOUT, SELECTABLE, Sequence } from "@/hardware/task/sequence/Sequence";
import { TYPE } from "@/hardware/task/sequence/types";
import { type Import } from "@/import";
import { type Layout } from "@/layout";
import { type Selector } from "@/selector";

export * from "@/hardware/task/sequence/palette";
export {
  createLayout,
  type CreateLayoutArgs,
  LAYOUT,
} from "@/hardware/task/sequence/Sequence";
export * from "@/hardware/task/sequence/types";

export const EXTRACTORS: Export.Extractors = { [TYPE]: Common.Task.extract };

export const FILE_INGESTORS: Import.FileIngestors = { [TYPE]: ingest };

export const LAYOUTS: Record<string, Layout.Renderer> = { [TYPE]: Sequence };

export const SELECTABLES: Selector.Selectable[] = [SELECTABLE];

export const ZERO_LAYOUTS: Record<string, Common.Task.Layout> = { [TYPE]: LAYOUT };
