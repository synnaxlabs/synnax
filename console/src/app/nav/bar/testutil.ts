// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, render, type RenderResult } from "@testing-library/react";
import { type ReactElement } from "react";

import { Session } from "@/session";
import { type ConsolePreloadedState, createConsoleWrapper } from "@/testutil";

export const client = createTestClient();

export const ACTIVE_PROJECT = await client.projects.create({
  name: "Ops",
  layout: {},
});

// withActiveProject seeds an active project, which the top bars require to render (in
// production they mount inside the project guard).
export const withActiveProject = (
  state: ConsolePreloadedState = {},
): ConsolePreloadedState => ({
  ...state,
  [Session.Project.SLICE_NAME]: {
    ...Session.Project.ZERO_SLICE_STATE,
    selected: ACTIVE_PROJECT.key,
  },
});

// renderBar mounts ui against a real client so access-gated nav items and the user/
// connection badges resolve through the production query path, and returns the render
// result plus the Redux store for asserting dispatched navigation actions.
//
// React holds a commit's passive effects while a sibling subtree is suspended, and the
// panel selector suspends. A synchronous mount returns before they flush, so no bar
// item's effects run. Browsers pump the scheduler on their own; a test has to.
export const renderBar = async (
  ui: ReactElement,
  preloadedState?: ConsolePreloadedState,
) => {
  const { wrapper, store } = await createConsoleWrapper({ client, preloadedState });
  let rendered!: RenderResult;
  await act(async () => {
    rendered = render(ui, { wrapper });
  });
  return { ...rendered, store };
};
