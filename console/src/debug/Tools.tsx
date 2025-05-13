// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Performance } from "@synnaxlabs/pluto";

import { type Layout } from "@/layout";

export const TOOLS_LAYOUT_TYPE = "debugTools";

export const TOOLS_LAYOUT: Layout.BaseState = {
  key: TOOLS_LAYOUT_TYPE,
  type: TOOLS_LAYOUT_TYPE,
  name: "Debug Tools",
  icon: "Channel",
  location: "mosaic",
};

export const Tools: Layout.Renderer = () => <Performance.Tools />;
