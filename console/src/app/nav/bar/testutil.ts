// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { render } from "@testing-library/react";
import { type ReactElement } from "react";

import { Project } from "@/project";
import { type ConsolePreloadedState, createConsoleWrapper } from "@/testUtils";

export const client = createTestClient();

// Live-core round-trips share the single test cluster with the rest of the suite, so
// allow more than the 1s waitFor default.
export const TIMEOUT = { timeout: 5000 };

export const ACTIVE_PROJECT = {
  key: "00000000-0000-0000-0000-000000000001",
  name: "Ops",
};

// withActiveProject seeds an active project, which the top bars require to render (in
// production they mount inside the project guard).
export const withActiveProject = (
  state: ConsolePreloadedState = {},
): ConsolePreloadedState => ({
  ...state,
  [Project.SLICE_NAME]: { ...Project.ZERO_SLICE_STATE, active: ACTIVE_PROJECT },
});

// renderBar mounts ui against a real client so access-gated nav items and the user/
// connection badges resolve through the production query path, and returns the render
// result plus the Redux store for asserting dispatched navigation actions.
export const renderBar = async (
  ui: ReactElement,
  preloadedState?: ConsolePreloadedState,
) => {
  const { wrapper, store } = await createConsoleWrapper({ client, preloadedState });
  return { ...render(ui, { wrapper }), store };
};
