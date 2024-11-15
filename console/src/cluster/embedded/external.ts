// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Controls } from "@/cluster/embedded/Controls";
import { controlsLayout } from "@/cluster/embedded/types";
import { type Layout } from "@/layout";
export * from "@/cluster/embedded/LogsProvider";
export * from "@/cluster/embedded/Toolbar";
export * from "@/cluster/embedded/use";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [controlsLayout.type]: Controls,
};
