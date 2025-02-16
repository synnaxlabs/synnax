// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { SELECTABLE, Sequence } from "@/hardware/task/sequence/Sequence";
import { TYPE } from "@/hardware/task/sequence/types";
import { type Layout } from "@/layout";

export * from "@/hardware/task/sequence/palette";
export {
  createLayout,
  type CreateLayoutArgs,
  LAYOUT,
} from "@/hardware/task/sequence/Sequence";
export * from "@/hardware/task/sequence/types";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [TYPE]: Sequence,
};

export const SELECTABLES: Layout.Selectable[] = [SELECTABLE];
