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
import { Session } from "@/session";
import {
  type ConsolePreloadedState,
  renderWithConsole,
  type TestStore,
} from "@/testutil";

export const createCluster = (
  key: string,
  overrides: Partial<Session.Node.Node> = {},
): Session.Node.Node => ({
  key,
  name: key,
  host: "localhost",
  port: 9090,
  username: "synnax",
  password: "seldon",
  secure: false,
  ...overrides,
});

export const createClusterState = (
  clusters: Session.Node.Node[],
  selected?: string,
): ConsolePreloadedState => ({
  [Session.Node.SLICE_NAME]: {
    version: 0,
    selected,
    nodes: Object.fromEntries(clusters.map((c) => [c.key, c])),
  },
});

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
