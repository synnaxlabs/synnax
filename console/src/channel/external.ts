// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Calculated, CALCULATED_LAYOUT_TYPE } from "@/channel/Calculated";
import { Create, CREATE_LAYOUT_TYPE } from "@/channel/Create";
import { type Layout } from "@/layout";

export * from "@/channel/Calculated";
export * from "@/channel/Create";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [CREATE_LAYOUT_TYPE]: Create,
  [CALCULATED_LAYOUT_TYPE]: Calculated,
};
