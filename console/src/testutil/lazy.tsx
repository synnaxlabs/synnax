// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type RenderOptions,
  type RenderResult,
  waitFor,
  type waitForOptions,
} from "@testing-library/react";
import { type ReactElement, Suspense, useEffect } from "react";

import { renderSuspended } from "@/testutil/testutil";

// A cold module load can outlast the default query wait, so the wait ends only short
// of the test timeout.
const LOAD_WAIT: waitForOptions = { timeout: 10_000 };

interface LoadProbeProps {
  onLoad: () => void;
}

// Commits with the rest of its Suspense boundary, so it runs once the boundary loads.
const LoadProbe = ({ onLoad }: LoadProbeProps): null => {
  useEffect(onLoad, [onLoad]);
  return null;
};

/**
 * Renders ui inside a Suspense boundary and resolves once the boundary leaves its
 * fallback, so queries run against loaded lazy components. Rejects when the boundary
 * does not load, with the first caught error as the cause.
 */
export const renderLazy = async (
  ui: ReactElement,
  options: RenderOptions = {},
): Promise<RenderResult> => {
  const { onCaughtError } = options;
  let loaded = false;
  let loadError: unknown;
  const result = await renderSuspended(
    <Suspense fallback={null}>
      {ui}
      <LoadProbe
        onLoad={() => {
          loaded = true;
        }}
      />
    </Suspense>,
    {
      ...options,
      onCaughtError: (error, info) => {
        loadError ??= error;
        // React logs a caught error when the root sets no handler.
        if (onCaughtError == null) console.error(error);
        else onCaughtError(error, info);
      },
    },
  );
  // waitFor runs outside act, since act holds React's retry until its callback returns.
  await waitFor(() => {
    if (loaded) return;
    throw new Error("The lazy component did not load", { cause: loadError });
  }, LOAD_WAIT);
  return result;
};
