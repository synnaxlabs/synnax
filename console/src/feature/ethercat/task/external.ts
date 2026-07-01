// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Read, READ_LAYOUT, ReadSelectable } from "@/feature/ethercat/task/Read";
import {
  READ_SCHEMAS,
  READ_TYPE,
  WRITE_SCHEMAS,
  WRITE_TYPE,
} from "@/feature/ethercat/task/types";
import { Write, WRITE_LAYOUT, WriteSelectable } from "@/feature/ethercat/task/Write";
import { createIngester } from "@/feature/task/createIngester";
import { extract } from "@/feature/task/export";
import { type Layout as TaskLayout } from "@/feature/task/Form";
import { type Export } from "@/platform/export";
import { type Import } from "@/platform/import";
import { type Layout } from "@/platform/layout";
import { type Selector } from "@/platform/selector";

export * from "@/feature/ethercat/task/palette";
export * from "@/feature/ethercat/task/Read";
export * from "@/feature/ethercat/task/types";
export * from "@/feature/ethercat/task/Write";

export const EXTRACTORS: Export.Extractors = {
  [READ_TYPE]: extract,
  [WRITE_TYPE]: extract,
};

export const FILE_INGESTERS: Import.FileIngesters = {
  [READ_TYPE]: createIngester(READ_SCHEMAS.config, READ_LAYOUT),
  [WRITE_TYPE]: createIngester(WRITE_SCHEMAS.config, WRITE_LAYOUT),
};

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [READ_TYPE]: Read,
  [WRITE_TYPE]: Write,
};

export const SELECTABLES: Selector.Selectable[] = [ReadSelectable, WriteSelectable];

export const ZERO_LAYOUTS: Record<string, TaskLayout> = {
  [READ_TYPE]: READ_LAYOUT,
  [WRITE_TYPE]: WRITE_LAYOUT,
};
