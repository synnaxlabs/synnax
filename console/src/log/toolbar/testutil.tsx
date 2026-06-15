// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type log } from "@synnaxlabs/client";
import { Log as PLog } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { act, render, within } from "@testing-library/react";
import {
  type ComponentType,
  type FC,
  type PropsWithChildren,
  type ReactElement,
  Suspense,
} from "react";

import { type ConsolePreloadedState, createConsoleWrapper } from "@/testUtils";

export const client = createTestClient();

// Live-core round-trips share the single test cluster with the rest of the suite, so
// allow more than the 1s waitFor default.
export const TIMEOUT = { timeout: 5000 };

let projectKey: string | undefined;
const project = async (): Promise<string> =>
  (projectKey ??= (await client.projects.create({ name: id.create(), layout: {} }))
    .key);

// loadLog primes key's flux cache through the production retrieve path. The single-hook
// bootstrap keeps the suspending useEnsureRetrieved from being followed by other hooks,
// a shape that trips a React 19 concurrent-replay error.
const loadLog = async (Wrapper: FC<PropsWithChildren>, key: string): Promise<void> => {
  const Bootstrap = (): ReactElement => {
    PLog.useEnsureRetrieved({ key });
    return <div data-testid="loaded" />;
  };
  let utils!: ReturnType<typeof render>;
  await act(async () => {
    utils = render(
      <Suspense fallback={null}>
        <Bootstrap />
      </Suspense>,
      { wrapper: Wrapper },
    );
  });
  await within(utils.container).findByTestId("loaded");
};

export interface RenderLogOptions {
  log?: Partial<log.New>;
  preloadedState?: (key: string) => ConsolePreloadedState;
}

// renderLog creates a log on the server, mounts Component with the log loaded into the
// flux cache, and returns the render result plus the Redux store and log key.
export const renderLog = async (
  Component: ComponentType<{ layoutKey: string }>,
  { log: logOverrides, preloadedState }: RenderLogOptions = {},
) => {
  const created = await client.logs.create(await project(), {
    name: "Test Log",
    ...logOverrides,
  });
  const { wrapper: Wrapper, store } = await createConsoleWrapper({
    client,
    preloadedState: preloadedState?.(created.key),
  });
  await loadLog(Wrapper, created.key);
  const result = render(<Component layoutKey={created.key} />, { wrapper: Wrapper });
  return { key: created.key, result, store };
};
