// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ConfigureRead, READ_SELECTABLE } from "@/hardware/labjack/task/Read";
import { READ_TYPE, WRITE_TYPE } from "@/hardware/labjack/task/types";
import { ConfigureWrite, WRITE_SELECTABLE } from "@/hardware/labjack/task/Write";
import { type Layout } from "@/layout";

export * from "@/hardware/labjack/task/common";
export * from "@/hardware/labjack/task/Read";
export * from "@/hardware/labjack/task/types";
export * from "@/hardware/labjack/task/Write";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [READ_TYPE]: ConfigureRead,
  [WRITE_TYPE]: ConfigureWrite,
};

export const SELECTABLES: Layout.Selectable[] = [READ_SELECTABLE, WRITE_SELECTABLE];
