// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Docs, LAYOUT_TYPE } from "@/platform/docs/Docs";
import { type Layout } from "@/platform/layout";

export * from "@/feature/docs/palette";
export * from "@/platform/docs/Docs";
export * from "@/session/docs/selectors";
export * from "@/session/docs/slice";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [LAYOUT_TYPE]: Docs,
};
