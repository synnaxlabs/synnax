// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Read, ReadSelectable } from "@/feature/opc/task/Read";
import { READ_TYPE, WRITE_TYPE } from "@/feature/opc/task/types";
import { Write, WriteSelectable } from "@/feature/opc/task/Write";
import { type Selector } from "@/platform/selector";
import { type Task } from "@/platform/task";

export * from "@/feature/opc/task/commands";
export * from "@/feature/opc/task/Read";
export * from "@/feature/opc/task/types";
export * from "@/feature/opc/task/Write";

export const FORMS: Task.Forms = {
  [READ_TYPE]: Read,
  [WRITE_TYPE]: Write,
};

export const SELECTABLES: Selector.Selectable[] = [ReadSelectable, WriteSelectable];
