// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type WindowState } from "@/window";

/** Creates a window state in the creating stage with the given props. */
export const createWindowState = (
  props: Partial<WindowState> & Pick<WindowState, "key">,
): WindowState => ({
  stage: "creating",
  reserved: false,
  focusCount: 0,
  centerCount: 0,
  ...props,
});
