// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type LayoutState } from "@/layout/layout";

export const createLayout = (overrides?: LayoutState): LayoutState => ({
  windowKey: "docs",
  key: "docs",
  type: "docs",
  location: "mosaic",
  name: "Documentation",
  tab: { editable: false },
  ...overrides,
});
