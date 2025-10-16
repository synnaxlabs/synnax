// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Export } from "@/export";
import { Common } from "@/hardware/common";
import { Read, READ_SELECTABLE } from "@/hardware/visa/task/Read";
import { READ_TYPE, WRITE_TYPE } from "@/hardware/visa/task/types";
import { Write, WRITE_SELECTABLE } from "@/hardware/visa/task/Write";
import { type Layout } from "@/layout";
import { type Selector } from "@/selector";

export * from "@/hardware/visa/task/Read";
export * from "@/hardware/visa/task/types";
export * from "@/hardware/visa/task/Write";

export const EXTRACTORS: Export.Extractors = {
  [READ_TYPE]: Common.Task.extract,
  [WRITE_TYPE]: Common.Task.extract,
};

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [READ_TYPE]: Read,
  [WRITE_TYPE]: Write,
};

export const SELECTABLES: Selector.Selectable[] = [READ_SELECTABLE, WRITE_SELECTABLE];

export const COMMANDS = [];