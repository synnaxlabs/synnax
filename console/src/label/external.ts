// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Edit, EDIT_LAYOUT_TYPE } from "@/label/Edit";
import { type Layout } from "@/layout";

export * from "@/label/Edit";
export * from "@/label/HasFilter";
export * from "@/label/Select";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [EDIT_LAYOUT_TYPE]: Edit,
};
