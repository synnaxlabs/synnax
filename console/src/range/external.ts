// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Layout } from "@/layout";
import { Create, CREATE_LAYOUT_TYPE } from "@/range/CreateLayout";
import { Overview, overviewLayout } from "@/range/overview/Overview";

export * from "@/range/ContextMenu";
export * from "@/range/CreateLayout";
export * from "@/range/overview/Overview";
export * from "@/range/Select";
export * from "@/range/selectors";
export * from "@/range/slice";
export * from "@/range/slice";
export * from "@/range/Toolbar";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [CREATE_LAYOUT_TYPE]: Create,
  [overviewLayout.type]: Overview,
};
