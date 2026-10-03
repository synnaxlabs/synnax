// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { vi } from "vitest";

import { Provider } from "@/platform/analytics/Context";
import { type Sink } from "@/platform/analytics/sink";

/** A sink whose every method is a spy, so a spec can read what was reported. */
export const createTestSink = () => ({
  capture: vi.fn<Sink["capture"]>(),
  screen: vi.fn<Sink["screen"]>(),
  identify: vi.fn<Sink["identify"]>(),
  reset: vi.fn<Sink["reset"]>(),
});

/** Returns the test wrapper with everything inside it reporting to the sink. */
export const wrapWithSink = (
  Wrapper: FC<PropsWithChildren>,
  sink: Sink,
): FC<PropsWithChildren> => {
  const Wrapped = ({ children }: PropsWithChildren): ReactElement => (
    <Wrapper>
      <Provider sink={sink}>{children}</Provider>
    </Wrapper>
  );
  Wrapped.displayName = "AnalyticsWrapper";
  return Wrapped;
};
