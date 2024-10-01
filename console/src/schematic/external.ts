// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Layout } from "@/layout";
import { LAYOUT_TYPE, Schematic } from "@/schematic/Schematic";

export * from "@/schematic/file";
export * from "@/schematic/middleware";
export * from "@/schematic/NavControls";
export * from "@/schematic/Schematic";
export * from "@/schematic/selectors";
export * from "@/schematic/slice";
export * from "@/schematic/toolbar";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [LAYOUT_TYPE]: Schematic,
};
