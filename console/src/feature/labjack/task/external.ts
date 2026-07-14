// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Read, readIngester, ReadSelectable } from "@/feature/labjack/task/Read";
import { READ_TYPE, WRITE_TYPE } from "@/feature/labjack/task/types";
import { Write, writeIngester, WriteSelectable } from "@/feature/labjack/task/Write";
import { type Export } from "@/platform/export";
import { type Import } from "@/platform/import";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

export * from "@/feature/labjack/task/commands";
export * from "@/feature/labjack/task/Read";
export * from "@/feature/labjack/task/SelectInputChannelTypeField";
export * from "@/feature/labjack/task/SelectOutputChannelType";
export * from "@/feature/labjack/task/types";
export * from "@/feature/labjack/task/Write";

export const EXTRACTORS: Export.Extractors = {
  [READ_TYPE]: Task.extract,
  [WRITE_TYPE]: Task.extract,
};

export const FILE_INGESTERS: Import.FileIngesters = {
  [READ_TYPE]: readIngester,
  [WRITE_TYPE]: writeIngester,
};

export const FORMS: Task.Forms = {
  [READ_TYPE]: Read,
  [WRITE_TYPE]: Write,
};

export const SELECTABLES: Selector.Selectable[] = [ReadSelectable, WriteSelectable];
