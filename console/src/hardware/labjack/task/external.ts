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
import { ingestRead, ingestWrite } from "@/hardware/labjack/task/import";
import { Read, READ_LAYOUT, ReadSelectable } from "@/hardware/labjack/task/Read";
import { READ_TYPE, WRITE_TYPE } from "@/hardware/labjack/task/types";
import { Write, WRITE_LAYOUT, WriteSelectable } from "@/hardware/labjack/task/Write";
import { type Import } from "@/import";
import { type Layout } from "@/layout";
import { type Selector } from "@/selector";

export * from "@/hardware/labjack/task/palette";
export * from "@/hardware/labjack/task/Read";
export * from "@/hardware/labjack/task/SelectInputChannelTypeField";
export * from "@/hardware/labjack/task/SelectOutputChannelType";
export * from "@/hardware/labjack/task/types";
export * from "@/hardware/labjack/task/Write";

export const EXTRACTORS: Export.Extractors = {
  [READ_TYPE]: Common.Task.extract,
  [WRITE_TYPE]: Common.Task.extract,
};

export const FILE_INGESTERS: Import.FileIngesters = {
  [READ_TYPE]: ingestRead,
  [WRITE_TYPE]: ingestWrite,
};

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [READ_TYPE]: Read,
  [WRITE_TYPE]: Write,
};

export const SELECTABLES: Selector.Selectable[] = [ReadSelectable, WriteSelectable];

export const ZERO_LAYOUTS: Record<string, Common.Task.Layout> = {
  [READ_TYPE]: READ_LAYOUT,
  [WRITE_TYPE]: WRITE_LAYOUT,
};
