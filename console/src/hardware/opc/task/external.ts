// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { READ_SELECTABLE, ReadTask } from "@/hardware/opc/task/Read";
import { READ_TYPE, WRITE_TYPE } from "@/hardware/opc/task/types/v0";
import { WRITE_SELECTABLE, WriteTask } from "@/hardware/opc/task/Write";
import { type Layout } from "@/layout";

export * from "@/hardware/opc/task/palette";
export * from "@/hardware/opc/task/Read";
export * from "@/hardware/opc/task/types";
export * from "@/hardware/opc/task/Write";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [READ_TYPE]: ReadTask,
  [WRITE_TYPE]: WriteTask,
};

export const SELECTABLES: Layout.Selectable[] = [READ_SELECTABLE, WRITE_SELECTABLE];
