// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { GetStarted } from "@/layout/GetStarted";
import { Renderer } from "@/layout/layout";
import { GET_STARTED_LAYOUT_TYPE } from "@/layout/slice";
import { Mosaic } from "@/layouts/mosaic";

export * from "@/layout/Content";
export * from "@/layout/context";
export * from "@/layout/GetStarted";
export * from "@/layout/layout";
export * from "@/layout/middleware";
export * from "@/layout/Window";
export * from "@/layout/selectors";
export * from "@/layout/palette";
export * from "@/layout/slice";
export * from "@/layout/hooks";

export const LAYOUTS: Record<string, Renderer> = {
  mosaic: Mosaic.Window,
  [GET_STARTED_LAYOUT_TYPE]: GetStarted,
};
