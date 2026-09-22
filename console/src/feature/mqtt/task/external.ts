// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Read, ReadSelectable } from "@/feature/mqtt/task/Read";
import { READ_TYPE, WRITE_TYPE } from "@/feature/mqtt/task/types";
import { Write, WriteSelectable } from "@/feature/mqtt/task/Write";
import { type Selector } from "@/platform/selector";
import { type Task } from "@/platform/task";

export * from "@/feature/mqtt/task/commands";
export * from "@/feature/mqtt/task/createReadFields";
export * from "@/feature/mqtt/task/Read";
export * from "@/feature/mqtt/task/sparkplug";
export * from "@/feature/mqtt/task/types";
export * from "@/feature/mqtt/task/Write";

export const FORMS: Task.Forms = {
  [READ_TYPE]: Read,
  [WRITE_TYPE]: Write,
};

export const SELECTABLES: Selector.Selectable[] = [ReadSelectable, WriteSelectable];
