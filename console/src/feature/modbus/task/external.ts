// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Read, READ_LAYOUT, ReadSelectable } from "@/feature/modbus/task/Read";
import {
  READ_SCHEMAS,
  READ_TYPE,
  WRITE_SCHEMAS,
  WRITE_TYPE,
} from "@/feature/modbus/task/types";
import { Write, WRITE_LAYOUT, WriteSelectable } from "@/feature/modbus/task/Write";
import { type Export } from "@/platform/export";
import { type Import } from "@/platform/import";
import { type Layout } from "@/platform/layout";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

export * from "@/feature/modbus/task/commands";
export * from "@/feature/modbus/task/Read";
export * from "@/feature/modbus/task/types";
export * from "@/feature/modbus/task/Write";

export const EXTRACTORS: Export.Extractors = {
  [READ_TYPE]: Task.extract,
  [WRITE_TYPE]: Task.extract,
};

export const FILE_INGESTERS: Import.FileIngesters = {
  [READ_TYPE]: Task.createIngester(READ_SCHEMAS.config, READ_LAYOUT),
  [WRITE_TYPE]: Task.createIngester(WRITE_SCHEMAS.config, WRITE_LAYOUT),
};

export const LAYOUTS: Layout.Renderers = {
  [READ_TYPE]: Read,
  [WRITE_TYPE]: Write,
};

export const SELECTABLES: Selector.Selectable[] = [ReadSelectable, WriteSelectable];

export const ZERO_LAYOUTS: Record<string, Task.Layout> = {
  [READ_TYPE]: READ_LAYOUT,
  [WRITE_TYPE]: WRITE_LAYOUT,
};
