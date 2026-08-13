// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type RenderResult } from "@testing-library/react";
import { type ReactNode } from "react";

import { Modals } from "@/platform/modals";
import {
  type ConsolePreloadedState,
  renderWithConsole,
  type TestStore,
} from "@/testutil";

/**
 * Renders ui together with a live {@link Modals.Stack} inside the full console provider
 * stack, so modals opened during the test actually mount. Returns the backing store so a
 * spec can inspect dispatched cluster state.
 */
export const renderClusterUI = async (
  ui: ReactNode,
  preloadedState?: ConsolePreloadedState,
): Promise<RenderResult & { store: TestStore }> =>
  await renderWithConsole(
    <>
      {ui}
      <Modals.Stack />
    </>,
    { preloadedState },
  );
