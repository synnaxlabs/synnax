// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Read, readIngester, ReadSelectable } from "@/feature/modbus/task/Read";
import { READ_TYPE, WRITE_TYPE } from "@/feature/modbus/task/types";
import { Write, writeIngester, WriteSelectable } from "@/feature/modbus/task/Write";
import { type Import } from "@/platform/import";
import { type Selector } from "@/platform/selector";
import { type Task } from "@/platform/task";

export * from "@/feature/modbus/task/commands";
export * from "@/feature/modbus/task/Read";
export * from "@/feature/modbus/task/types";
export * from "@/feature/modbus/task/Write";

export const FILE_INGESTERS: Import.FileIngesters = {
  [READ_TYPE]: readIngester,
  [WRITE_TYPE]: writeIngester,
};

export const FORMS: Task.Forms = {
  [READ_TYPE]: Read,
  [WRITE_TYPE]: Write,
};

export const SELECTABLES: Selector.Selectable[] = [ReadSelectable, WriteSelectable];
