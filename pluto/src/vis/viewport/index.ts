// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useViewport } from "./useViewPort";
import { ViewportMask } from "./ViewportMask";
export type {
  UseViewportProps,
  UseViewportReturn,
  UseViewportTriggers,
  UseViewportHandler,
  UseViewportEvent,
} from "./useViewPort";
export type { ViewportMaskProps } from "./ViewportMask";

export const Viewport = {
  use: useViewport,
  Mask: ViewportMask,
};
