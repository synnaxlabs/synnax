// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Read, READ_SELECTABLE } from "@/hardware/opc/task/Read";
import { READ_TYPE, WRITE_TYPE } from "@/hardware/opc/task/types";
import { Write, WRITE_SELECTABLE } from "@/hardware/opc/task/Write";
import { type Layout } from "@/layout";

export * from "@/hardware/opc/task/palette";
export * from "@/hardware/opc/task/Read";
export * from "@/hardware/opc/task/types";
export * from "@/hardware/opc/task/Write";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [READ_TYPE]: Read,
  [WRITE_TYPE]: Write,
};

export const SELECTABLES = [READ_SELECTABLE, WRITE_SELECTABLE];
