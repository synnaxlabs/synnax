// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  act,
  render,
  type RenderOptions,
  type RenderResult,
  waitFor,
  type waitForOptions,
} from "@testing-library/react";
import {
  type PropsWithChildren,
  type ReactElement,
  Suspense,
  useEffect,
  useLayoutEffect,
  useState,
} from "react";

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

interface GateProps extends PropsWithChildren {
  onMount: (show: () => void) => void;
}

// Shows its children once the show function it hands out is called.
const Gate = ({ onMount, children }: GateProps): ReactElement | null => {
  const [shown, setShown] = useState(false);
  useLayoutEffect(() => onMount(() => setShown(true)), [onMount]);
  return shown ? <>{children}</> : null;
};

/**
 * Renders ui inside a Suspense boundary and resolves once the boundary leaves its
 * fallback, so queries run against loaded lazy components. Rejects when the boundary
 * does not load, with the first caught error as the cause.
 *
 * A tree wired to a live Core turns the act environment off, and its boundary mounts
 * outside act. An async act returns only once the page has no work left, which such a
 * tree can fail to reach before the test times out.
 */
export const renderLazy = async (
  ui: ReactElement,
  options: RenderOptions = {},
): Promise<RenderResult> => {
  const { onCaughtError } = options;
  let loaded = false;
  let loadError: unknown;
  const gate: { show?: () => void } = {};
  const result = render(
    <Gate
      onMount={(show) => {
        gate.show = show;
      }}
    >
      <Suspense fallback={null}>
        {ui}
        <LoadProbe
          onLoad={() => {
            loaded = true;
          }}
        />
      </Suspense>
    </Gate>,
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
  if (gate.show == null)
    throw new Error("The wrapper of the lazy component did not mount");
  const { show } = gate;
  if (globalThis.IS_REACT_ACT_ENVIRONMENT === true) await act(async () => show());
  else show();
  await waitFor(() => {
    if (loaded) return;
    throw new Error("The lazy component did not load", { cause: loadError });
  }, LOAD_WAIT);
  return result;
};
