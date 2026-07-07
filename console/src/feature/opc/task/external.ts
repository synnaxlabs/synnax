// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Read, ReadSelectable } from "@/feature/opc/task/Read";
import {
  READ_SCHEMAS,
  READ_TYPE,
  WRITE_SCHEMAS,
  WRITE_TYPE,
} from "@/feature/opc/task/types";
import { Write, WriteSelectable } from "@/feature/opc/task/Write";
import { type Export } from "@/platform/export";
import { type Import } from "@/platform/import";
import { type Panel } from "@/platform/panel";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

export * from "@/feature/opc/task/palette";
export * from "@/feature/opc/task/Read";
export * from "@/feature/opc/task/types";
export * from "@/feature/opc/task/Write";

export const EXTRACTORS: Export.Extractors = {
  [READ_TYPE]: Task.extract,
  [WRITE_TYPE]: Task.extract,
};

export const FILE_INGESTERS: Import.FileIngesters = {
  [READ_TYPE]: Task.createIngester(READ_SCHEMAS.config, READ_TYPE),
  [WRITE_TYPE]: Task.createIngester(WRITE_SCHEMAS.config, WRITE_TYPE),
};

export const TABS: Panel.Tabs = {
  [READ_TYPE]: Read,
  [WRITE_TYPE]: Write,
};

export const SELECTABLES: Selector.Selectable[] = [ReadSelectable, WriteSelectable];
